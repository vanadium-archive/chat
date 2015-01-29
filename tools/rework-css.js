var autoprefixer = require('autoprefixer');
var fs = require('fs');
var path = require('path');

var rework = require('rework');
var colors = require('rework-plugin-colors');
var imprt = require('rework-import');
var vars = require('rework-vars');

var infile = process.argv[2];
if (!infile) {
  throw new Error('Usage: node rework-css.js <infile>');
}

var css = fs.readFileSync(infile, 'utf8');
css = rework(css)
  .use(imprt({path: path.dirname(infile)}))
  .use(vars())
  .use(colors())
  .toString();
css = autoprefixer().process(css).css;
process.stdout.write(css);
