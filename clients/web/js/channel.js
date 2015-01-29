module.exports = Channel;

var _ = require('lodash');
var EventEmitter = require('events').EventEmitter;
var nutil = require('veyron').namespaceUtil;
var path = require('path');
var streamToArray = require('stream-to-array');

var noop = require('./noop');
var util = require('./util');

function memberNames(members) {
  return _.map(members, function(x) {
    return nutil.basename(x.name);
  }).sort();
}

// Joins the specified channel, creating it if needed.
// Emits 'members' and 'message' events.
function Channel(rt, channelName, userName, cb) {
  cb = cb || noop;

  this.channelName_ = path.join('apps/chat', channelName);

  this.namespace_ = rt.namespace();
  this.context_ = rt.getContext();
  this.client_ = rt.newClient();
  this.server_ = rt.newServer();

  this.ee_ = new EventEmitter();
  this.members_ = [];
  this.intervalID_ = null;  // initialized below

  var that = this;

  var service = {
    sendMessage: function(ctx, text) {
      that.ee_.emit('message', {
        sender: util.firstShortName(ctx.remoteBlessingStrings),
        text: text,
        timestamp: new Date()
      });
    }
  };

  // Note, serve() performs the mount() for us.
  var serviceName = path.join(this.channelName_, userName);
  // TODO(nlacasee,sadovsky): Our current authorization policy never returns any
  // errors, i.e. everyone is authorized!
  var openAuthorizer = function(){ return null; };
  var options = {authorizer: openAuthorizer};
  this.server_.serve(serviceName, service, options, function(err) {
    if (err) return cb(err);
    // Use nextTick() for the first updateMembers_() call to give clients a
    // chance to set up their event listeners.
    process.nextTick(that.updateMembers_.bind(that));
    // TODO(sadovsky): Replace with mounttable glob watch.
    that.intervalID_ = setInterval(that.updateMembers_.bind(that), 2000);
    return cb();
  });
}

Channel.prototype.leave = function(cb) {
  cb = cb || noop;
  clearInterval(this.intervalID_);
  // TODO(sadovsky): Provide a public method to stop the server.
  this.server_.stop(cb);
};

Channel.prototype.broadcastMessage = function(messageText, cb) {
  var that = this;
  cb = cb || noop;
  // Schedule all messages to be sent, then return immediately.
  // TODO(sadovsky): Better error handling, perhaps?
  _.forEach(this.members_, function(member) {
    process.nextTick(function() {
      var ctx = that.context_.withTimeout(5000);
      that.client_.bindTo(ctx, member.name, function(err, s) {
        ctx.cancel();
        if (err) {
          console.error(err);
        } else {
          var ctx2 = that.context_.withTimeout(5000);
          s.sendMessage(ctx2, messageText, function(err) {
            if (err) console.error(err);
            ctx2.cancel();
          });
        }
      });
    });
  });
  return cb();
};

Channel.prototype.updateMembers_ = function() {
  var that = this;

  var ctx = this.context_.withTimeout(5000);
  var pattern = path.join(this.channelName_, '*');
  var globStream = this.namespace_.glob(ctx, pattern).stream;

  streamToArray(globStream, function(err, members) {
    ctx.cancel();
    if (err) throw err;

    var oldMemberNames = memberNames(that.members_);
    that.members_ = members;
    var newMemberNames = memberNames(that.members_);
    if (!_.isEqual(oldMemberNames, newMemberNames)) {
      that.ee_.emit('members', newMemberNames);
    }
  });
};

Channel.prototype.addEventListener = function(event, listener) {
  this.ee_.addListener(event, listener);
  return this;
};

Channel.prototype.on = Channel.prototype.addEventListener;

Channel.prototype.removeEventListener = function(event, listener) {
  this.ee_.removeListener(event, listener);
  return this;
};

Channel.prototype.off = Channel.prototype.removeEventListener;
