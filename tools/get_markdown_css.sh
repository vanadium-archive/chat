# Copyright 2015 The Vanadium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -e
set -u

TMPDIR="${TMPDIR-/tmp}"

main() {
  local -r TMP=$(mktemp -d "${TMPDIR}/XXXXXX")
  local -r OUTFILE="${TMP}/markdown-preview.css"

  cd $TMP
  echo "Downloading files to ${TMP}"

  curl -L -O https://raw.githubusercontent.com/atom/markdown-preview/master/stylesheets/markdown-preview.less
  curl -L -O https://raw.githubusercontent.com/atom/template-syntax/master/stylesheets/colors.less
  curl -L -O https://raw.githubusercontent.com/atom/template-syntax/master/stylesheets/syntax-variables.less

  npm install less
  ./node_modules/.bin/lessc markdown-preview.less > "${OUTFILE}"

  echo "Wrote ${OUTFILE}"
}

main "$@"
