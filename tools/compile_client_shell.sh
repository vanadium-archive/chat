#!/bin/bash

# Compiles the shell client for specified target architecture.  Returns an error
# if host architecture does not support comilation to target.

# NOTE(nlacasse): Cross-compilation across different OS's is nearly impossible
# because of Veyron's dependency on CGO.
#
# I installed a golang cross-compiler that works with CGO, as described here:
# https://gist.github.com/steeve/6905542
# (These instructions are slightly out-of-date.  It is not necessary to do the
# "Apply the patch" step.  Instead, you must run "go-cross-compile-build-all"
# with environment variable "CGO_ENABLED=1".)
#
# However, those instructions depend on "a valid toolchain for the platform/os
# you're targetting [sic]".  Installing a Darwin toolchain on Linux is not easy,
# and may not even be possible.  The most recent instructions I found
# (https://github.com/Tatsh/xchain) were over two years old, and required
# extracting gcc header files from an (even older) XCode dmg, then patching and
# compiling gcc manually.
#
# I *was* able to compile a version of chat by hacking out Veyron's dependency
# on CGO.  Specifically, I removed the NewNetConfigWatcher() function in
# veyron/lib/netconfig/ipaux_bsd.go, stubbed out the Platform() function in
# veyron/profiles/platform_darwin.go, and removed the "import C" from both of
# those files.  Compiliation was then easy, and only required a golang
# cross-compiler.  However, this is not really a viable release strategy, and
# will probably stop working in the future, as more Veyron components depend on
# CGO.  Some of the Veyron code that uses CGO is in the roaming support, which
# I imagine many clients (including chat) will want to use.

source "${VEYRON_ROOT}/scripts/lib/shell.sh"

export GOPATH="$(pwd)/clients/shell"
export VDLPATH="$(pwd)/clients/shell"

at_exit() {
  shell::at_exit
  shell::kill_child_processes
}

usage() {
  echo "Usage: `basename $0` <darwin|linux>"
  echo "Compiles shell client for specified target architecture."
  echo "Returns an error if host architecture does not support compilation to target."
  exit 1
}

err_unsupported_compilation() {
  echo "Error: Cannot compile for target $1 on host $2."
  exit 1
}

precompile_tasks() {
  veyron goext distclean
  # TODO(sadovsky): This is only valid if we are not cross-compiling.
  make build-shell
}

compile_darwin() {
  precompile_tasks
  # The Veyron cross-compiler tool "veyron xgo" does not support the
  # amd64-darwin profile, so just use regular compiler "veyron go build".
  veyron go build -o clients/shell/bin/dist/chat-darwin-amd64/chat client
}

compile_linux() {
  precompile_tasks
  veyron xgo amd64-linux build -o clients/shell/bin/dist/chat-linux-amd64/chat client
}

main() {
  local -r OS=$(uname)

  if [[ $# -ne 1 ]]; then
    usage
  elif [[ $1 != "darwin" && $1 != "linux" ]]; then
    usage
  elif [[ $1 == "darwin" && ${OS} == "Darwin" ]]; then
    compile_darwin
  elif [[ $1 == "linux" && ${OS} == "Linux" ]]; then
    compile_linux
  else
    err_unsupported_compilation $1 ${OS}
  fi
  exit 0
}

main "$@"
