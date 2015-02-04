SHELL := /bin/bash -euo pipefail
export PATH := node_modules/.bin:clients/shell/bin:$(PATH)
export GOPATH := $(shell pwd)/clients/shell:$(GOPATH)
export VDLPATH := $(GOPATH)

# Don't use VANADIUM_ROOT if NO_VANADIUM_ROOT is set.
# This is equivalent to "VANADIUM_ROOT= make ..."
ifdef NO_VANADIUM_ROOT
  VANADIUM_ROOT :=
endif

# If VANADIUM_ROOT is defined, we should compile/build our clients against the
# code there.  This allows us to test our clients againts the current code, and
# simplifies debugging.  In order to make this work, we must change our PATHs
# and go compiler depending on whether VANADIUM_ROOT is set.
ifdef VANADIUM_ROOT
	# Use "v23" go compiler wrapper.
	GO := v23 go
	# v23 puts binaries in $(VANADIUM_ROOT)/release/go/bin, so add that to the PATH.
	PATH := $(VANADIUM_ROOT)/release/go/bin:$(PATH)
	# Add location of node and npm from environment repo.
	export PATH := $(VANADIUM_ROOT)/environment/cout/node/bin:$(PATH)
else
	# Use standard go compiler.
	GO := go
	# The vdl tool needs either VANADIUM_ROOT or VDLROOT, so set VDLROOT.
	export VDLROOT := $(shell pwd)/clients/shell/src/v.io/core/veyron2/vdl/vdlroot
endif


# This target causes any target files to be deleted if the target task fails.
# This is especially useful for browserify, which creates files even if it
# fails.
.DELETE_ON_ERROR:

UNAME := $(shell uname)

# When running browser tests on non-Darwin machines, set the --headless flag.
# This uses Xvfb underneath the hood (inside prova => browser-launcher =>
# headless), which is not supported on OS X.
# See: https://github.com/kesla/node-headless/
ifndef NOHEADLESS
	ifneq ($(UNAME),Darwin)
		HEADLESS := --headless
	endif
endif

ifdef STOPONFAIL
	STOPONFAIL := --stopOnFirstFailure
endif

ifndef NOTAP
	TAP := --tap
endif

ifndef NOQUIT
	QUIT := --quit
endif

ifdef XUNIT
	TAP := --tap # TAP must be set for xunit to work
	OUTPUT_TRANSFORM := tap-xunit
endif

ifdef BROWSER_OUTPUT
	BROWSER_OUTPUT_LOCAL = $(BROWSER_OUTPUT)
	ifdef OUTPUT_TRANSFORM
		BROWSER_OUTPUT_LOCAL := >($(OUTPUT_TRANSFORM) --package=javascript.browser > $(BROWSER_OUTPUT_LOCAL))
	endif
	BROWSER_OUTPUT_LOCAL := | tee $(BROWSER_OUTPUT_LOCAL)
endif

PROVA_OPTS := --includeFilenameAsPackage $(TAP) $(QUIT) $(STOPONFAIL)

BROWSER_OPTS := --browser --launch chrome $(HEADLESS) --log=./tmp/chrome.log

all: build-shell build-web

node_modules: package.json
	npm prune
	npm install
ifdef VANADIUM_ROOT
	# If VANADIUM_ROOT is defined, link veyron.js from it.
	rm -rf ./node_modules/veyron
	cd "$(VANADIUM_ROOT)/release/javascript/core" && npm link
	npm link veyron
else
	# If VANADIUM_ROOT is not defined, install veyron.js from github.
	npm install git+ssh://git@github.com:veyron/veyron.js.git
endif
	touch node_modules

# TODO(sadovsky): Make it so we only run "go install" when binaries are out of
# date.
veyron-binaries: clients/shell/src/v.io
	$(GO) install \
	v.io/core/veyron2/vdl/vdl \
	v.io/core/veyron/services/mounttable/mounttabled \
	v.io/core/veyron/tools/{principal,servicerunner}

clients/shell/src/github.com/fatih/color:
	$(GO) get github.com/fatih/color

clients/shell/src/github.com/kr/text:
	$(GO) get github.com/kr/text

clients/shell/src/github.com/nlacasse/gocui:
	$(GO) get github.com/nlacasse/gocui

clients/shell/src/v.io:
# Only go get v.io go repo if VANADIUM_ROOT is not defined.
ifndef VANADIUM_ROOT
	$(GO) get v.io/core/...
endif

clients/shell/bin/chat: veyron-binaries
clients/shell/bin/chat: clients/shell/src/github.com/fatih/color
clients/shell/bin/chat: clients/shell/src/github.com/kr/text
clients/shell/bin/chat: clients/shell/src/github.com/nlacasse/gocui
clients/shell/bin/chat: $(shell find clients/shell/src -name "*.go")
	vdl generate --lang=go chat/vdl
	$(GO) install chat

build-shell: veyron-binaries clients/shell/bin/chat

mkdir-build:
	@mkdir -p build

build/bundle.css: clients/web/css/index.css $(shell find clients/web/css -name "*.css") mkdir-build node_modules
	node tools/rework-css.js $< 1> $@

# https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html
# Note, on OS X you'll likely need to run "ulimit -S -n 1024" for the browserify
# command to succeed. (Run "ulimit -S -a" and "ulimit -H -a" to see all soft and
# hard limits respectively.)
# Also see: https://github.com/substack/node-browserify/issues/899
build/bundle.js: clients/web/js/index.js $(shell find clients/web/js -name "*.js") mkdir-build node_modules
#browserify $< -d -p [minifyify --map $(@F).map --output $@.map] -o $@
	browserify $< -d -o $@

build/index.html: clients/web/public/index.html mkdir-build
	cp $< $@

build/markdown-preview.css: markdown/markdown-preview.css mkdir-build
	cp $< $@

# This task has the minimal set of dependencies to build the web client assets,
# so that it can be run on a GCE instance during the deploy process.
# In particular, it does not depend on a veyron environment or golang.
build-web-assets: mkdir-build node_modules build/bundle.css build/bundle.js build/index.html build/markdown-preview.css $(shell find markdown -name "*.md")
	node tools/render-md.js

# TODO(sadovsky): For some reason, browserify and friends get triggered on each
# build-web invocation, even if their inputs haven't changed.
build-web: build-web-assets veyron-binaries

serve-web: build-web-assets
	node server.js

test: test-shell test-web

test-shell: build-shell
	# We must pass --veyron.tcp.address=localhost:0, otherwise the chat server
	# will listen on the external IP address of the gce instance, and our
	# firewall rules prevent connections on unknown ports unless coming from
	# localhost.
	$(GO) test chat/... --veyron.tcp.address=localhost:0

# We use the same test runner as veyron.js.  It handles starting and stopping
# all required services (proxy, wspr, mounntabled), and runs tests in chrome
# with prova.
# TODO(sadovsky): Some of the deps in our package.json are needed solely for
# runner.js. We should restructure things so that runner.js is its own npm
# package with its own deps.
test-web: lint build-web
ifndef VANADIUM_ROOT
	# The js tests needs the extension built into a folder so that it can be
	# loaded with chrome on startup.  The extension build process currently
	# depends on v23, the Vanadium "web" profile, and VANADIUM_ROOT.
	#
	# TODO(nlacasse): Either make the extension build process have less
	# dependencies, or distribute a version of the extension that can be
	# unpacked into a directory and used in tests by other projects like chat.
	@echo "The test-web make task requires VANADIUM_ROOT to be set."
	exit 1
else
	node ./node_modules/veyron/test/integration/runner.js -- \
	make test-web-runner
endif

# Note: runner.js sets the NAMESPACE_ROOT and PROXY_ADDR env vars for the
# spawned test subprocess; we specify "make test-web-runner" as the test
# command so that we can then reference these vars in the Vanadium extension
# and our prova command.
test-web-runner: APP_FRAME := "./build/index.html?mtname=$(NAMESPACE_ROOT)"
test-web-runner: VANADIUM_JS := $(VANADIUM_ROOT)/release/javascript/core
test-web-runner: BROWSER_OPTS := --options="--load-extension=$(VANADIUM_JS)/extension/build-test/,--ignore-certificate-errors,--enable-logging=stderr" $(BROWSER_OPTS)
test-web-runner:
	@$(RM) -fr $(VANADIUM_JS)/extension/build-test
	$(MAKE) -C $(VANADIUM_JS)/extension build-test
	prova clients/web/test/test-*.js -f $(APP_FRAME) $(PROVA_OPTS) $(BROWSER_OPTS) $(BROWSER_OUTPUT_LOCAL)

clean:
	rm -rf node_modules
	rm -rf clients/shell/{bin,pkg,src/code.google.com,src/github.com,src/golang.org,src/v.io}
	rm -rf build
	rm -rf veyron.js

lint: node_modules
	jshint .

.PHONY: all veyron-binaries go-deps
.PHONY: build-shell build-web-assets build-web serve-web
.PHONY: test test-shell test-web clean lint

.NOTPARALLEL: test
