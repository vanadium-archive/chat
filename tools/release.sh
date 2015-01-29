#!/bin/bash

# Deploys web site and shell client binaries, and tags release with version.

source "${VEYRON_ROOT}/scripts/lib/shell.sh"

trap at_exit INT TERM EXIT

at_exit() {
  # Note: shell::at_exit unsets our trap, so it won't run again on exit.
  shell::at_exit
  shell::kill_child_processes
}

# Checks that local master branch has been merged into veyron upstream.
check_branch_merged_to_veyron_remote() {
  git fetch --tags veyron master
  if [[ -n $(git diff --name-only remotes/veyron/master) ]]; then
    echo "Current branch contains changes that are not in veyron/master."
    echo "Please merge branch into veyron/master before releasing."
    exit 1
  fi
}

# Checks that remote named "veyron" exists.
check_veyron_remote_exists() {
  git remote show veyron > /dev/null
  if [[ $? -ne 0 ]]; then
    echo "Branch 'veyron' does not exist."
    echo "Please run: git remote add veyron git@github.com:veyron/chat.git"
    exit 1
  fi
}

# Checks that version has semver format like "v1.2.3".
check_valid_version() {
  if [[ ! $1 =~ ^v[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+$ ]]; then
    usage
  fi
}

usage() {
  echo "Usage: `basename $0` <version>"
  echo "version must begin with 'v' followed by a semver version, e.g. v1.2.3"
  echo ""
  echo "Released versions are:"
  git tag -l -n1
  echo ""
  exit 1
}

main() {
  if [[ $# -ne 1 ]]; then
    usage
  fi

  check_veyron_remote_exists
  check_branch_merged_to_veyron_remote
  check_valid_version $1

  # Deploy first, to make sure it succeeds.
  ./tools/deploy.sh all

  # Tag release and push tags to veyron remote.
  # TODO(nlacasse): Make sure the version is higher than all other version tags.
  git tag -a $1 -m "Release $1"
  git push veyron master --tags
}

main "$@"
