var _ = require('lodash');
var genColor = require('color-generator');
var moment = require('moment');
var React = require('react');

var Channel = require('./channel');
var h = require('./react_h');

// Navigation bar.
var NavBar = React.createClass({
  render: function() {
    return h('div.navbar', [
      h('span.title', document.title)
    ]);
  }
});

var userColors = {};

// Shows a single message.
var Message = React.createClass({
  render: function() {
    var m = this.props.message;
    userColors[m.sender] = (
      userColors[m.sender] || genColor(0.5, 0.75).hexString());
    return h('div.message', [
      h('span.timestamp', [
        moment(m.timestamp.toISOString()).format('MMM D [at] h:mma')
      ]),
      h('span.sender', {
        style: {color: userColors[m.sender]},
      }, m.sender),
      h('span.text', m.text)
    ]);
  }
});

// Shows received messages.
var Messages = React.createClass({
  componentDidUpdate: function() {
    var el = this.getDOMNode();
    el.scrollTop = el.scrollHeight;
  },
  render: function() {
    return h('div.messages', _.map(this.props.messages, function(v, i) {
      return new Message({key: i, message: v});
    }));
  }
});

// Shows a list of channel members.
var Members = React.createClass({
  render: function() {
    return h('div.members', [
      h('span.title', 'Members'),
      h('ul', _.map(this.props.members, function(member) {
        return h('li', {key: member}, [
          // http://graphemica.com/%E2%97%8F
          h('span.status-active', {key: 'status'}, '\u25CF'),
          h('span', {key: 'member'}, member)
        ]);
      }))
    ]);
  }
});

// Message entry box.
var Compose = React.createClass({
  getInput: function() {
    return this.getDOMNode().querySelector('input');
  },
  componentDidMount: function() {
    this.getInput().focus();
  },
  render: function() {
    var that = this;
    return h('div.compose', h('form', {
      onSubmit: function() {
        try {
          var input = that.getInput();
          that.props.broadcastMessage(input.value);
          // TODO(sadovsky): Maybe wait to reset input until the message is
          // received by the sender?
          input.value = '';
        } catch (e) {
          // TODO(sadovsky): For some reason, "throw e" doesn't work.
          console.error(e);
        } finally {
          return false;
        }
      }
    }, [
      h('input', {type: 'text', size: 128, placeholder: 'Enter message'}),
      // Invisible button used by the tests to submit the form.
      h('button', {type: 'submit', hidden: true})
    ]));
  }
});

var Page = React.createClass({
  getInitialState: function() {
    return {
      chan: null,
      members: null,
      messages: null
    };
  },
  componentWillReceiveProps: function(nextProps) {
    var that = this;
    var rt = nextProps.rt;
    if (!rt) return;

    var chan = null;
    // Note: Even if we somehow fail to call chan.leave(), our server will get
    // removed from the mounttable after one minute (due to ttl).
    window.addEventListener('beforeunload', function() {
      if (chan) chan.leave();
    });
    chan = new Channel(rt, 'public');
    chan.on('members', function(members) {
      that.setState({members: members});
    }).on('message', function(message) {
      that.setState({messages: that.state.messages.concat([message])});
    });
    this.setState({
      chan: chan,
      members: [],
      messages: []
    });
  },
  render: function() {
    var that = this;
    if (this.props.err) {
      return h('div', [
        h('span.alert-error', '' + this.props.err),
        h('div.instructions', [
          'Follow ',
          h('a', {href: 'help.html'}, 'these instructions'),
          ' to install and run Vanadium Chat.'
        ])
      ]);
    } else if (!this.props.rt) {
      return h('span.alert-info', 'Loading...');
    }
    return h('div.page', [
      new NavBar(),
      h('div.messages-members', [
        new Messages({messages: this.state.messages}),
        new Members({members: this.state.members})
      ]),
      new Compose({
        broadcastMessage: function(messageText) {
          that.state.chan.broadcastMessage(messageText, function(err) {
            if (err) throw err;
          });
        }
      })
    ]);
  }
});

////////////////////////////////////////
// Exports

module.exports = {
  Page: Page,
};
