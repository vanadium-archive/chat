var React = require('react');
var url = require('url');
var vanadium = require('vanadium');

var Page = require('./components').Page;

var u = url.parse(window.location.href, true);
var vanadiumConfig = {
  logLevel: vanadium.vlog.levels.INFO,
  namespaceRoots: u.query.mounttable ? [u.query.mounttable] : undefined,
  proxy: u.query.proxy
};

var page = React.renderComponent(
  new Page({rt: null}), document.querySelector('#c'));

// Export page on the window for testing/debugging.
window.page = page;

vanadium.init(vanadiumConfig, function(err, rt) {
  if (err) {
    if (err instanceof vanadium.errors.ExtensionNotInstalledError) {
      return vanadium.extension.promptUserToInstallExtension();
    }
    return displayError(err);
  }

  rt.on('crash', displayError);

  page.setProps({rt: rt});
});

function displayError(err) {
  console.error(err);
  page.setProps({err: err});
}
