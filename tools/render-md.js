// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var _ = require('lodash');
var cons = require('consolidate');
var fs = require('fs');
var glob = require('glob');
var marked = require('marked');
var path = require('path');

_.each(glob.sync('README.md'), function(infile) {
  var data = fs.readFileSync(infile);
  var body = marked(data.toString('binary'));
  var basename = path.basename(infile, '.md');
  cons.mustache('markdown/template.html', {
    body: body,
    title: basename
  }, function(err, html) {
    if (err) throw err;
    var outfile = path.join('build', basename + '.html');
    fs.writeFileSync(outfile, html);
    console.log(infile, '->', outfile);
  });
});
