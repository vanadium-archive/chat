module.exports = {
  shortName: shortName,
  firstShortName: firstShortName
};

// Note, shortName and firstShortName are duplicated between JS and Go.

// TODO(sadovsky): Fix mismatch between names on received messages and names
// mounted in the mount table. Perhaps our unit of operation should be blessing
// rather than client instance, and multiple client instances using the same
// blessing should be treated as multiple connections for the same user (similar
// to Hangouts with phone and desktop).

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
  if (blessings.length === 0) {
    return 'unknown';
  }
  for (var i = 0; i < blessings.length; i++) {
    var sn = shortName(blessings[i]);
    if (sn) return sn;
  }
  return blessings[0];
}
