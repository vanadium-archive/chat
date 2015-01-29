var React = require('react');
var url = require('url');
var veyron = require('veyron');

var Page = require('./components').Page;

var u = url.parse(window.location.href, true);
var veyronConfig = {
  logLevel: veyron.logLevels.INFO,
  authenticate: u.query.skipauth === undefined
};

var mtname = u.query.mtname || '/proxy.envyor.com:8101';

var page = React.renderComponent(
  new Page({rt: null}), document.querySelector('#c'));

veyron.init(veyronConfig, function(err, rt) {
  if (err) return displayError(err);

  rt.namespace().setRoots(mtname, function(err) {
    if (err) return displayError(err);
    page.setProps({rt: rt});
  });
});

function displayError(err) {
  console.error(err);
  page.setProps({err: err});
}
