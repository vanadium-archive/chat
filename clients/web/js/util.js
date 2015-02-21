module.exports = {
  shortName: shortName,
  firstShortName: firstShortName,
  randomHex: randomHex
};

// Note, shortName and firstShortName are duplicated between JS and Go.

function shortName(fullName) {
  // Split into components and see if any is an email address. A very
  // sophisticated technique is used to determine if the component is an email
  // address: presence of an "@" character.
  var parts = fullName.split('/');  // security.ChainSeparator
  for (var j = 0; j < parts.length; j++) {
    var p = parts[j];
    if (p.indexOf('@') > 0) {
      return p;
    }
  }
  return '';
}

function firstShortName(blessings) {
  if (!blessings || blessings.length === 0) {
    return 'unknown';
  }
  for (var i = 0; i < blessings.length; i++) {
    var sn = shortName(blessings[i]);
    if (sn) return sn;
  }
  return blessings[0];
}

function randomBytes(len) {
  len = len || 1;
  var array = new Int8Array(len);
  window.crypto.getRandomValues(array);
  return new Buffer(array);
}

function randomHex(len) {
  return randomBytes(Math.ceil(len/2)).toString('hex').substr(0, len);
}
