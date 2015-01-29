#!/bin/bash

# Compiles binaries, seeks blessings and starts wsprd for web client.

source "${VEYRON_ROOT}/scripts/lib/shell.sh"

trap at_exit INT TERM EXIT

at_exit() {
  # Note: shell::at_exit unsets our trap, so it won't run again on exit.
  shell::at_exit
  shell::kill_child_processes
}

usage() {
  echo "Usage: `basename $0` [hostname]"
  exit 1
}

main() {
  if [[ $# -gt 1 ]]; then
    usage
  fi

  veyron go install veyron.io/veyron/veyron/tools/principal veyron.io/wspr/veyron/services/wsprd

  local -r VEYRON_BIN="${VEYRON_ROOT}/veyron/go/bin"

  HOSTNAME="proxy.envyor.com"
  IDENTD_MT_PATH="identity/veyron-test"
  IDENTD_PROVIDER="google"

  if [[ $# -eq 1 ]]; then
    echo "Non-envyor hostname not yet supported"
    exit 1

    # TODO(sadovsky): Fix IDENTD_{MT_PATH,PROVIDER}.
    HOSTNAME="$1"
    IDENTD_MT_PATH="identity/foo"
    IDENTD_PROVIDER="bar"
  fi

  local -r MOUNTTABLED="${HOSTNAME}:8101"
  local -r PROXYD="${HOSTNAME}:8100"
  local -r IDENTD="${HOSTNAME}:8125"

  local -r IDENTD_NAME="/${MOUNTTABLED}/${IDENTD_MT_PATH}/${IDENTD_PROVIDER}"

  # Generate credentials for WSPR if they don't already exist.
  local -r CREDENTIALS_WSPRD="/tmp/chat.credentials.${HOSTNAME}"
  if [[ ! -d "${CREDENTIALS_WSPRD}" ]]; then
    # TODO(ashankar,sadovsky): Ideally, WSPR's credentials should not matter.
    # Figure out why we need this and possibly remove this whole step!
    VEYRON_CREDENTIALS="${CREDENTIALS_WSPRD}" "${VEYRON_BIN}/principal" seekblessings
  fi

  VEYRON_CREDENTIALS="${CREDENTIALS_WSPRD}" "${VEYRON_BIN}/wsprd" --identd="${IDENTD_NAME}" --veyron.proxy="${PROXYD}" --v=1 --alsologtostderr=true

  # TODO(sadovsky): Open the chat app url using xdg-open or somesuch.

  tail -f /dev/null  # block until Ctrl-C
}

main "$@"
