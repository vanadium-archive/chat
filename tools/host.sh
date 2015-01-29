#!/bin/bash

# Generate identities (for identityd and wsprd) and starts daemons (including
# wsprd) for chat app host.

# TODO(sadovsky): Broken until we support configuring trusted roots.
# See detailed TODO below.

source "${VEYRON_ROOT}/scripts/lib/shell.sh"

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

  make veyron-binaries

  local -r VEYRON_BIN="${VEYRON_ROOT}/veyron/go/bin"

  # Note, we make these services public so that fellow backpackers will be able
  # to connect to us. (We're sitting around a campfire, remember?)
  local -r MOUNTTABLED_ADDR=":8101"
  local -r PROXYD_ADDR=":8100"
  local -r IDENTD_ADDR=":8125"

  # TODO(sadovsky): Fix IDENTD_{MT_PATH,PROVIDER}.
  local -r HOSTNAME="$(hostname)"
  local -r IDENTD_MT_PATH="identity/foo"
  local -r IDENTD_PROVIDER="bar"

  local -r MOUNTTABLED="${HOSTNAME}:8101"
  local -r PROXYD="${HOSTNAME}:8100"
  local -r IDENTD="${HOSTNAME}:8125"

  local -r IDENTD_NAME="/${MOUNTTABLED}/${IDENTD_MT_PATH}/${IDENTD_PROVIDER}"

  "${VEYRON_BIN}/mounttabled" --veyron.tcp.address="${MOUNTTABLED_ADDR}" --v=1 --alsologtostderr=true &

  "${VEYRON_BIN}/proxyd" --address="${PROXYD_ADDR}" --v=1 --alsologtostderr=true &

  # Generate a self-signed identity to run identityd as.
  local -r CREDENTIALS_IDENTD=$(shell::tmp_dir)
  "${VEYRON_BIN}/principal" create "${CREDENTIALS_IDENTD}"

  # TODO(sadovsky): Run a fake identityd that does not require network access to
  # Google to issue blessings, and update crx to support auth flows that do not
  # involve Google.
  VEYRON_CREDENTIALS="${CREDENTIALS_IDENTD}" "${VEYRON_BIN}/identityd" -httpaddr="${IDENTD_ADDR}" -host="${HOSTNAME}" -vaddr="${PROXYD_ADDR}" -google_config_chrome="./tools/google_config_chrome.json" &

  # TODO(ashankar,sadovsky): See TODO above. Though, WSPRs credentials shouldn't
  # matter. Figure out why they do and skip this step.
  local -r CREDENTIALS_WSPRD=$(shell::tmp_dir)
  VEYRON_CREDENTIALS="${CREDENTIALS_WSPRD}" "${VEYRON_BIN}/principal" seekblessings

  VEYRON_CREDENTIALS="${CREDENTIALS_WSPRD}" "${VEYRON_BIN}/wsprd" --identd="${IDENTD_NAME}" --veyron.proxy="${PROXYD}" --v=1 --alsologtostderr=true &  # listens on port 8124

  make build-web-assets
  node ./server.js
}

main "$@"
