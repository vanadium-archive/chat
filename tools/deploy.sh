#!/bin/bash

# Builds and uploads web site assets and shell client binaries.
# If you want to create a new release tag, use release.sh instead.

# TODO(nlacasse): Make the deploy server location a parameter.

source "${VEYRON_ROOT}/scripts/lib/shell.sh"

trap at_exit INT TERM EXIT

at_exit() {
  # Note: shell::at_exit unsets our trap, so it won't run again on exit.
  shell::at_exit
  shell::kill_child_processes
}

# Compile binary for host OS and push to staging.
deploy_shell() {
  local -r OS=$(uname)
  if [[ $OS =~ "Darwin" ]]; then
    echo "Building for Darwin"
    ./tools/compile_client_shell.sh darwin
  elif [[ $OS =~ "Linux" ]]; then
    echo "Building for Linux"
    ./tools/compile_client_shell.sh linux
  else
    echo "Unknown OS: $OS"
    exit 1
  fi

  # Copy binaries to staging.v.io, and set the permissions so they can
  # be downloaded.
  rsync -avz --chmod=u+rwx,g+rx,o+rx clients/shell/bin/dist/* git@staging.v.io:/usr/share/nginx/chat/binaries
}

# Build and deploy the web assets.
deploy_web() {
  git push staging master
}

usage() {
  echo "Usage `basename $0` <shell|web|all>"
  echo "  shell: Compile and push shell binaries."
  echo "  web: Compile and push web assets."
  echo "  all: Compile and push shell binaries and web assets."
  exit 1
}

main() {
  # Make sure anything we deploy is built from scratch, just in case our
  # Makefile setup doesn't always rebuild things that it should (which has
  # happened in the past).
  make clean
  if [[ $# -ne 1 ]]; then
    usage
  elif [[ $1 == "all" ]]; then
    deploy_shell
    deploy_web
  elif [[ $1 == "shell" ]]; then
    deploy_shell
  elif [[ $1 == "web" ]]; then
    deploy_web
  else
    usage
  fi
  exit 0
}

main "$@"
