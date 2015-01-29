#!/bin/bash

# Compiles binaries, generates credentials, and runs the shell client.

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

  veyron go install veyron.io/veyron/veyron/tools/principal

  # TODO(nlacasse): This assumes the user is building the client from source,
  # and is in the root of the repo.  How will this work with pre-built binaries?
  make build-shell

  local -r VEYRON_BIN="${VEYRON_ROOT}/veyron/go/bin"

  # TODO(nlacasse): Support alternative "hostname" argument.
  local -r HOSTNAME="proxy.envyor.com"
  local -r MOUNTTABLED="${HOSTNAME}:8101"
  local -r PROXYD="${HOSTNAME}:8100"
  local -r IDENTD="${HOSTNAME}:8125"
  local -r IDENTD_MT_PATH="identity/veyron-test"
  local -r IDENTD_PROVIDER="google"

  # Generate credentials if they don't exist.
  export VEYRON_CREDENTIALS="/tmp/chat.identity.${HOSTNAME}"
  if [[ ! -d "${VEYRON_CREDENTIALS}" ]]; then
    "${VEYRON_BIN}/principal" seekblessings
  fi

  # TODO(nlacasse): How will this work with pre-built binaries?
  ./clients/shell/bin/client --veyron.proxy="${PROXYD}"
}

main "$@"
