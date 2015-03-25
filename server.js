// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Static web server, for delivering the web client application bundle.

var http = require('http');
var path = require('path');
var st = require('st');

var argv = require('minimist')(process.argv.slice(2));
var hostname = argv.hostname || 'localhost';
var port = argv.port || 4000;

var server = http.createServer(st({
  index: 'index.html',
  path: path.join(__dirname, 'build'),
  url: '/',
  // Turn off caching for development.
  cache: false
})).listen(port, hostname, function() {
  var addr = server.address();
  console.log('Serving http://%s:%d', addr.address, addr.port);
});
