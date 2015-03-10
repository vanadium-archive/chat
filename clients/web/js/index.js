var React = require('react');
var url = require('url');
var vanadium = require('vanadium');

var Page = require('./components').Page;

var u = url.parse(window.location.href, true);
var vanadiumConfig = {
  logLevel: vanadium.vlog.levels.INFO,
  authenticate: u.query.skipauth === undefined
};

var mtname = u.query.mtname || '/proxy.envyor.com:8101';

var page = React.renderComponent(
  new Page({rt: null}), document.querySelector('#c'));

// Export page on the window for testing/debugging.
window.page = page;

vanadium.init(vanadiumConfig, function(err, rt) {
  if (err) return displayError(err);

  rt.on('error', displayError);

  rt.namespace().setRoots(mtname, function(err) {
    if (err) return displayError(err);
    page.setProps({rt: rt});
  });
});

function displayError(err) {
  console.error(err);
  page.setProps({err: err});
}
