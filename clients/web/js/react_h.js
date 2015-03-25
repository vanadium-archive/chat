// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Mercury-like h() function that constructs React virtual DOM nodes.

var _ = require('lodash');
var React = require('react');

module.exports = function(selector, properties, children) {
  if (!_.isPlainObject(properties)) {
    children = properties;
    properties = {};
  } else {
    console.assert(!properties.id && !properties.className);
  }
  var parts = selector.split('.');
  var x = parts[0].split('#'), type = x[0], id = x[1];
  var className = parts.slice(1).join(' ');
  properties = _.assign({}, properties, {id: id, className: className});
  return React.DOM[type](properties, children);
};
