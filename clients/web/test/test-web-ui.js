var domready = require('domready');
var test = require('prova');

// Simulates a click event on an element.
function click(elt) {
  if (!elt) {
    throw new Error('Error calling "click": element is undefined.');
  }
  elt.dispatchEvent(new MouseEvent('click', {
    view: window,
    bubbles: true,
    cancelable: true
  }));
}

// Checks that an element contains text.
function contains(elt, text) {
  if (!elt) {
    throw new Error('Error calling "contains": element is undefined.');
  }
  return elt.innerHTML.indexOf(text) >= 0;
}

test('Basic UI elements', function(t) {
  domready(function() {
    // TODO(nlacasse): These setTimeouts are a bummer, but it can take a while
    // for veyron.init and setting the namespace root.  Come up with something
    // better.
    setTimeout(function() {
      t.ok(contains(document.body, 'Vanadium Chat'), 'header exists');
      t.ok(contains(document.body, 'Enter message'), 'message input exists');
      t.end();
    }, 3000);
  });
});

test('Members list', function(t) {
  domready(function() {
    setTimeout(function() {
      var membersList = document.querySelector('.members ul');
      // NOTE(nlacasse): We check for *at least* one member (as opposed to
      // *exactly* one member) because if you run the tests multiple times in
      // the same browser instance, the old members stick around until they time
      // out of the mounttable, which causes multiple members to be in the list
      // temporarily.
      t.ok(membersList.children.length >= 1, 'at least one member in list');
      t.end();
    }, 3000);
  });
});

test('Sending a message', function(t) {
  domready(function() {
    setTimeout(function() {
      var form = document.querySelector('.compose form');
      var input = form.querySelector('input');
      var button = form.querySelector('button');
      var messages = document.querySelector('.messages');

      var newMessage = 'Hello Vanadium world!';
      input.value = newMessage;

      // NOTE(nlacasse): I tried using React TestUtils to submit the form with
      // Simulate.submit(form) and Simululate.keyPress(input, {key: 'Enter'}),
      // but neither method caused the onSubmit handler the fire.  The only
      // thing I could make work was to create a hidden 'submit' button, and
      // send a fake click there.
      click(button);

      t.equal(input.value, '', 'input is empty after sending message');

      // Give the message time to be received.
      setTimeout(function() {
        t.ok(contains(messages, newMessage),
             'message is contained in message list');
        t.end();
      }, 1000);
    });
  }, 3000);
});
