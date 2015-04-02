#!/bin/bash
# Copyright 2015 The Vanadium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.


# Generate identities and starts daemons for chat app host.
# TODO(nlacasse): Consider re-writing this in Go.

source $VANADIUM_ROOT/release/projects/chat/tools/shell.sh

trap at_exit INT TERM EXIT

at_exit() {
  # Note: shell::at_exit unsets our trap, so it won't run again on exit.
  shell::at_exit
  shell::kill_child_processes
}

usage() {
  echo "Usage: `basename $0`"
  exit 1
}

main() {
  if [[ $# -ne 0 ]]; then
    usage
  fi

  make vanadium-binaries

  local -r VANADIUM_BIN="${VANADIUM_ROOT}/release/go/bin"

  # Generate a self-signed identity to run identityd as.
  local -r VANADIUM_CREDENTIALS=$(shell::tmp_dir)
  "${VANADIUM_BIN}/principal" seekblessings --v23.credentials "${VANADIUM_CREDENTIALS}"

  local -r PROXYD_ADDR="localhost:8100"
  local -r MOUNTTABLED_ADDR="localhost:8101"

  "${VANADIUM_BIN}/mounttabled" --v23.tcp.address="${MOUNTTABLED_ADDR}" \
      --v23.credentials="${VANADIUM_CREDENTIALS}" \
      --v23.tcp.protocol=ws \
      --v=1 --alsologtostderr=true &

  # Give the mounttable time to start.
  sleep 2

  "${VANADIUM_BIN}/proxyd" --v23.namespace.root="/${MOUNTTABLED_ADDR}" \
      --v23.credentials="${VANADIUM_CREDENTIALS}" \
      --v23.tcp.address="${PROXYD_ADDR}" \
      --name=proxy \
      --v=1 --alsologtostderr=true &

  # Wait forever.
  sleep infinity
}

main "$@"
