// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

module.exports = Channel;

var _ = require('lodash');
var EventEmitter = require('events').EventEmitter;
var inherits = require('inherits');
var path = require('path');

var access = require('vanadium/src/gen-vdl/v.io/v23/services/security/access');
var noop = require('./noop');
var ServiceVdl = require('./chat/vdl');
var util = require('./util');

// Member is a member of the channel.
function Member(blessings, path) {
  // The member's blessings.
  this.blessings = blessings;
  // The name of the member.
  this.name = util.firstShortName(blessings);
  // The path at which the member is mounted in the mounttable.
  this.path = path;
}

// memberNames takes an array of members and returns a sorted list of this
// names.
function memberNames(members) {
  return _.map(members, function(member) {
    return member.name;
  }).sort();
}

// Channel encapsulates the logic for a client of the Vanadium Chat.  It
// inherits from EventEmitter and emits 'members', 'message', and 'ready'
// events.
function Channel(rt, channelName) {
  EventEmitter.call(this);

  this.channelName_ = path.join('apps/chat', channelName);

  this.accountName_ = rt.accountName;
  this.namespace_ = rt.namespace();
  this.context_ = rt.getContext();
  this.client_ = rt.newClient();
  this.server_ = rt.newServer();

  this.ready_ = false;
  this.members_ = [];
  this.intervalID_ = null;
}

inherits(Channel, EventEmitter);

// join creates a Vanadium server and mounts it in the mounttable under a
// random "locked" name.
Channel.prototype.join = function(cb) {
  cb = cb || noop;

  // Create our service implementation, which defines a single method:
  // "SendMessage".  We inherit from ServiceVdl.Chat which causes our defined
  // VDL types to be used when serializing values on the wire.  This is
  // necessary for the web client to be able to communicate with the shell
  // client.
  var Service = function() {
    ServiceVdl.Chat.call(this);
  };
  inherits(Service, ServiceVdl.Chat);

  // The implementation of sendMessage emits the message with the sender's
  // name and timestamp.
  Service.prototype.sendMessage = function(ctx, text) {
    that.emit('message', {
      sender: util.firstShortName(ctx.remoteBlessingStrings),
      text: text,
      timestamp: new Date()
    });
  };

  // openAuthorizer allows RPCs from all clients.
  var openAuthorizer = function(){ return null; };
  var options = {authorizer: openAuthorizer};

  var that = this;

  // Get a locked name to mount under.
  this.getLockedName_(function(err, name) {
    if (err) {
      return cb(err);
    }

    that.mountedName_ = name;

    // Serve the chat service under the locked name.
    // Note, serve() performs the mount() for us.
    that.server_.serve(name, new Service(), options, function(err) {
      if (err) return cb(err);
      that.updateMembers_();
      that.intervalID_ = setInterval(that.updateMembers_.bind(that), 2000);
      return cb();
    });
  });
};

// getLockedName picks a random name and attempts to "lock" it by setting
// restrictive permissions.  It tries repeatedly until it picks a name that is
// not locked by another client.
Channel.prototype.getLockedName_ = function(cb) {
  // openACL gives everybody permission.
  var openACL = new access.AccessList({
    'in': ['...']
  });

  // myACL only gives my blessings and decendants permission.
  var myACL = new access.AccessList({
    'in': [this.accountName_]
  });

  // Create a tagged acl map with the desired permissions.
  var tam = new access.Permissions(new Map([
    // Give everybody the ability to read and resolve the name.
    ['Read', openACL],
    ['Resolve', openACL],
    // All other permissions are only for us.
    ['Admin', myACL],
    ['Create', myACL],
    ['Mount', myACL]
  ]));

  var that = this;

  // Repeatedly pick random names and try to setPermissions on them until we get
  // one that has not already been locked.
  var maxRetries = 25;
  function attemptToGetName(tries) {
    if (tries >= maxRetries) {
      return cb(new Error('Tried ' + maxRetries + ' to get an unlocked name ' +
            'but did not succeed.'));
    }

    // Choose a random name under the channel name.
    var name = path.join(that.channelName_, util.randomHex(32));
    var ctx = that.context_.withTimeout(5000);
    that.namespace_.setPermissions(ctx, name, tam, '', function(err) {
      ctx.done();
      if (err) {
        // Try again.
        return attemptToGetName(tries + 1);
      }
      return cb(null, name);
    });
  }

  attemptToGetName(0);
};

// leave stops the chat server, and deletes our name from the mounttable.
Channel.prototype.leave = function(cb) {
  cb = cb || noop;

  // Stop updating member names.
  if (this.intervalID_) {
    clearInterval(this.intervalID_);
    this.intervalID_ = null;
  }

  // Delete our name from the mounttable.
  if (this.mountedName_) {
    this.namespace_.delete(this.context_, this.mountedName_, true, noop);
  }

  // Stop the server.
  this.server_.stop(cb);
};

// sendMessageTo sends a message to a particular member.
Channel.prototype.sendMessageTo = function(member, messageText, cb) {
  cb = cb || noop;

  // The allowedServersPolicy options require that the server matches the
  // blessings we got when we globbed it.
  var callOpts = this.client_.callOptions({
    allowedServersPolicy: member.blessings
  });

  var ctx = this.context_.withTimeout(5000);

  // Bind to the member's chat server.
  this.client_.bindTo(ctx, member.path, function(err, s) {
    if (err) {
      ctx.done();
      return cb(err);
    }

    // Invoke sendMessage on the member's chat server with messageText.
    s.sendMessage(ctx, messageText, callOpts, function(err) {
      if (err) {
        return cb(err);
      }
      ctx.done();
      return cb(null);
    });
  });
};

// broadcastMessage sends a message to all members in the channel.
Channel.prototype.broadcastMessage = function(messageText, cb) {
  cb = cb || noop;
  var that = this;

  // Schedule all messages to be sent, then return immediately.
  _.forEach(this.members_, function(member) {
    process.nextTick(function() {
      that.sendMessageTo(member, messageText);
    });
  });
  return cb(null);
};

// updateMembers_ globs the mounttable and constructs member objects for each
// server in the glob results.  Channel will emit 'members' event if the
// members have changed since the last glob.
Channel.prototype.updateMembers_ = function() {
  var that = this;

  // We glob on the channel name to find all the members in our channel.  The
  // glob will return a stream of mount entries corresponding to the paths where
  // channel members are mounted.  Then, we get the remote blessings from each
  // member, and use those as the member names.

  var ctx = this.context_.withTimeout(1000);
  var pattern = path.join(this.channelName_, '*');

  // Start the glob rpc.  This returns a promise with a "stream" property.
  // Mount entries are emitted on the stream.
  var globRpc = this.namespace_.glob(ctx, pattern);
  var globStream = globRpc.stream;
  globRpc.catch(function(err) {
    console.error(err);
  });

  var newMembers = [];

  // Each time we get a mount entry, we construct a new Member object.
  globStream.on('data', function(mountEntry) {
    if (!mountEntry.servers || mountEntry.servers.length === 0) {
      // No servers mounted at that name, only a lonely ACL.  Safe to ignore.
      return;
    }

    var blessings = mountEntry.servers[0].blessingPatterns;
    var path = mountEntry.name;

    newMembers.push(new Member(blessings, path));
  });

  globStream.on('end', done);

  function done(err) {
    if (err) {
      console.error(err);
    }

    if (newMembers.length === 0) {
      // No glob results, not even us!  Don't emit anything yet.
      return;
    }

    // Get the member names for the old and new members and if they differ
    // emit a "members" event which the UI will use to update the members
    // list.
    var oldMemberNames = memberNames(that.members_);
    that.members_ = newMembers;
    var newMemberNames = memberNames(that.members_);
    if (!_.isEqual(oldMemberNames, newMemberNames)) {
      that.emit('members', newMemberNames);
    }

    if (!that.ready_) {
      that.ready_ = true;
      that.emit('ready');
    }
  }
};
