SHELL := /bin/bash -euo pipefail
PATH := node_modules/.bin:clients/shell/bin:$(PATH)
export GOPATH := $(shell pwd)/clients/shell:$(GOPATH)
export VDLPATH := $(GOPATH)

# TODO(nlacasse): Use the code in $VANADIUM_ROOT when testing so we can catch
# errors caused by dependencies changing.  It would be nice to maintain a
# "standalone" build process like we have now too.

# This is needed for web tests, to build the Vanadium extension.
VANADIUM_JS:=$(VANADIUM_ROOT)/release/javascript/core

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
	touch node_modules

# TODO(sadovsky): Make it so we only run "go install" when binaries are out of
# date.
veyron-binaries: clients/shell/src/v.io
	go install \
	v.io/core/veyron2/vdl/vdl \
	v.io/core/veyron/tools/{mounttable,principal,servicerunner,vrpc}

clients/shell/src/github.com/fatih/color:
	go get github.com/fatih/color

clients/shell/src/github.com/kr/text:
	go get github.com/kr/text

clients/shell/src/github.com/nlacasse/gocui:
	go get github.com/nlacasse/gocui

clients/shell/src/v.io:
	go get v.io/core/...

clients/shell/bin/client: veyron-binaries
clients/shell/bin/client: clients/shell/src/github.com/fatih/color
clients/shell/bin/client: clients/shell/src/github.com/kr/text
clients/shell/bin/client: clients/shell/src/github.com/nlacasse/gocui
clients/shell/bin/client: $(shell find clients/shell/src -name "*.go")
	vdl generate --lang=go service
	go install client

build-shell: veyron-binaries clients/shell/bin/client

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
	go test client/...

# We use the same test runner as veyron.js.  It handles starting and stopping
# all required services (proxy, wspr, mounntabled), and runs tests in chrome
# with prova.
# TODO(sadovsky): Some of the deps in our package.json are needed solely for
# runner.js. We should restructure things so that runner.js is its own npm
# package with its own deps.
test-web: lint build-web
	node ./node_modules/veyron/test/integration/runner.js -- \
	make test-web-runner

# Note: runner.js sets the NAMESPACE_ROOT and PROXY_ADDR env vars for the
# spawned test subprocess; we specify "make test-web-runner" as the test
# command so that we can then reference these vars in the Vanadium extension
# and our prova command.
test-web-runner: APP_FRAME := "./build/index.html?mtname=$(NAMESPACE_ROOT)"
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
