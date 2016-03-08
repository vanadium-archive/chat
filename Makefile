SHELL := /bin/bash -euo pipefail

NODE_DIR := $(shell jiri profile list --info Target.InstallationDir v23:nodejs)
export PATH := $(JIRI_ROOT)/release/go/bin:node_modules/.bin:$(NODE_DIR)/bin:clients/shell/go/bin:$(PATH)
export GOPATH := $(shell pwd)/clients/shell/go:$(GOPATH)
export VDLPATH := $(shell pwd)/clients/shell/go/src:$(JIRI_ROOT)/release/go/src
GO := jiri go

# This target causes any target files to be deleted if the target task fails.
# This is especially useful for browserify, which creates files even if it
# fails.
.DELETE_ON_ERROR:

# Default browserify options: use sourcemaps.
BROWSERIFY_OPTS := --debug
# Names that should not be mangled by minification.
RESERVED_NAMES := 'context,ctx,callback,cb,$$stream,serverCall'
# Don't mangle RESERVED_NAMES, and screw ie8.
MANGLE_OPTS := --mangle [--except $(RESERVED_NAMES) --screw_ie8 ]
# Don't remove unused variables from function arguments, which could mess up signatures.
# Also don't evaulate constant expressions, since we rely on them to conditionally require modules only in node.
COMPRESS_OPTS := --compress [ --no-unused --no-evaluate ]
# Work-around for Browserify opening too many files by increasing the limit on file descriptors.
# https://github.com/substack/node-browserify/issues/431
INCREASE_FILE_DESC = ulimit -S -n 2560

# Browserify and extract sourcemap, but do not minify.
define BROWSERIFY
	mkdir -p $(dir $2)
	$(INCREASE_FILE_DESC); \
	browserify $1 $(BROWSERIFY_OPTS) | exorcist $2.map > $2
endef

# Browserify, minify, and extract sourcemap.
define BROWSERIFY-MIN
	mkdir -p $(dir $2)
	$(INCREASE_FILE_DESC); \
	browserify $1 $(BROWSERIFY_OPTS) --g [ uglifyify $(MANGLE_OPTS) $(COMPRESS_OPTS) ] | exorcist $2.map > $2
endef

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

.DEFAULT_GOAL := all

.PHONY: all
all: build-shell build-web

.PHONY: deploy-production
deploy-production: build-web-assets
	make -C $(JIRI_ROOT)/infrastructure/deploy chat-production

.PHONY: deploy-staging
deploy-staging: build-web-assets
	make -C $(JIRI_ROOT)/infrastructure/deploy chat-staging

node_modules: package.json
	npm prune
	npm install
	# Link Vanadium from JIRI_ROOT.
	rm -rf ./node_modules/vanadium
	cd "$(JIRI_ROOT)/release/javascript/core" && npm link
	npm link vanadium
	touch node_modules

# TODO(sadovsky): Make it so we only run "go install" when binaries are out of
# date.
vanadium-binaries:
	$(GO) install -a -tags wspr v.io/x/ref/cmd/servicerunner
	$(GO) install \
	v.io/x/ref/services/mounttable/mounttabled \
	v.io/x/ref/services/xproxy/xproxyd \
	v.io/x/ref/cmd/{principal,vdl} \
	v.io/x/ref/services/agent/agentd

gen-vdl: vanadium-binaries
	vdl generate --lang=go v.io/x/chat/vdl
	vdl generate --lang=javascript --js-out-dir=clients/web/js v.io/x/chat/vdl

clients/shell/go/bin/chat: vanadium-binaries gen-vdl
clients/shell/go/bin/chat: $(shell find clients/shell/go/src -name "*.go")
	$(GO) install v.io/x/chat

build-shell: vanadium-binaries clients/shell/go/bin/chat

run-shell: build-shell
	clients/shell/run.sh

mkdir-build:
	@mkdir -p build

build/bundle.css: clients/web/css/index.css $(shell find clients/web/css -name "*.css") mkdir-build node_modules
	node tools/rework-css.js $< 1> $@

# https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html
# Note, on OS X you'll likely need to run "ulimit -S -n 1024" for the browserify
# command to succeed. (Run "ulimit -S -a" and "ulimit -H -a" to see all soft and
# hard limits respectively.)
# Also see: https://github.com/substack/node-browserify/issues/899
build/bundle.js: clients/web/js/index.js $(shell find clients/web/js -name "*.js") gen-vdl mkdir-build node_modules vanadium-binaries
ifndef NOMINIFY
	$(call BROWSERIFY,$<,$@)
else
	$(call BROWSERIFY-MIN,$<,$@)
endif

build/index.html: clients/web/public/index.html mkdir-build
	cp $< $@

build/markdown-preview.css: markdown/markdown-preview.css mkdir-build
	cp $< $@

# This task has the minimal set of dependencies to build the web client assets,
# so that it can be run on a GCE instance during the deploy process.
# In particular, it does not depend on a vanadium environment or golang.
build-web-assets: mkdir-build node_modules build/bundle.css build/bundle.js build/index.html build/markdown-preview.css README.md
	node tools/render-md.js

# TODO(sadovsky): For some reason, browserify and friends get triggered on each
# build-web invocation, even if their inputs haven't changed.
build-web: build-web-assets vanadium-binaries

serve-web: build-web-assets
	node server.js

test: test-shell test-web

test-shell: build-shell
	# We must pass --v23.tcp.address=localhost:0, otherwise the chat server
	# will listen on the external IP address of the gce instance, and our
	# firewall rules prevent connections on unknown ports unless coming from
	# localhost.
	$(GO) test v.io/x/chat/... --v23.tcp.address=localhost:0

# We use the same test runner as vanadium.js.  It handles starting and stopping
# all required services (proxy, wspr, mounntabled), and runs tests in chrome
# with prova.
# TODO(sadovsky): Some of the deps in our package.json are needed solely for
# runner.js. We should restructure things so that runner.js is its own npm
# package with its own deps.
test-web: lint build-web
	node ./node_modules/vanadium/test/integration/runner.js -- \
	make test-web-runner

# Note: runner.js sets the V23_NAMESPACE and PROXY_ADDR env vars for the
# spawned test subprocess; we specify "make test-web-runner" as the test
# command so that we can then reference these vars in the Vanadium extension
# and our prova command.
test-web-runner: APP_FRAME := "./build/index.html?mtname=$(V23_NAMESPACE)"
test-web-runner: VANADIUM_JS := $(JIRI_ROOT)/release/javascript/core
test-web-runner: BROWSER_OPTS := --options="--load-extension=$(VANADIUM_JS)/extension/build-test/,--ignore-certificate-errors,--enable-logging=stderr" $(BROWSER_OPTS)
test-web-runner:
	$(MAKE) -C $(VANADIUM_JS)/extension clean
	$(MAKE) -C $(VANADIUM_JS)/extension build-test
	prova clients/web/test/test-*.js -f $(APP_FRAME) $(PROVA_OPTS) $(BROWSER_OPTS) $(BROWSER_OUTPUT_LOCAL)


# Run UI tests for the chat web client.
# These tests do not normally need to be run locally, but they can be if you
# want to verify that the a specific version of chat is compatible with a
# local (or live) version of the Vanadium extension.
#
# This test takes additional environment variables (typically temporary)
# - GOOGLE_BOT_USERNAME and GOOGLE_BOT_PASSWORD (To sign into Google/Chrome)
# - CHROME_WEBDRIVER (The path to the chrome web driver)
# - WORKSPACE (optional, defaults to $JIRI_ROOT/release/projects/chat)
# - TEST_URL (optional, defaults to https://chat.staging.v.io)
# - NO_XVFB (optional, defaults to using Xvfb. Set to true to watch the test.)
# - BUILD_EXTENSION (optional, defaults to using the live one. Set to true to
#                    use a local build of the Vanadium extension.)
#
# In addition, this test requires that maven, Xvfb, and xvfb-run be installed.
# The HTML report will be in $JIRI_ROOT/release/projects/chat/htmlReports
WORKSPACE ?= $(JIRI_ROOT)/release/projects/chat
TEST_URL ?= https://chat.staging.v.io
ifndef NO_XVFB
	XVFB := TMPDIR=/tmp xvfb-run -s '-ac -screen 0 1024x768x24'
endif

ifdef BUILD_EXTENSION
	BUILD_EXTENSION_PROPERTY := "-DvanadiumExtensionPath=$(VANADIUM_JS)/extension/build"
endif

test-ui:
ifdef BUILD_EXTENSION
	make -B -C $(VANADIUM_JS)/extension build-dev
endif
	WORKSPACE=$(WORKSPACE) $(XVFB) \
	  mvn test \
	  -f=$(JIRI_ROOT)/release/projects/chat/clients/web/test/ui/pom.xml \
	  -Dtest=ChatUITest \
	  -DchromeDriverBin=$(CHROME_WEBDRIVER) \
	  -DhtmlReportsRelativePath=htmlReports \
	  -DgoogleBotUsername=$(GOOGLE_BOT_USERNAME) \
	  -DgoogleBotPassword=$(GOOGLE_BOT_PASSWORD) \
	  $(BUILD_EXTENSION_PROPERTY) \
	  -DtestUrl=$(TEST_URL)

clean:
	rm -rf build
	rm -rf clients/shell/go/{bin,pkg}
	rm -rf clients/shell/credentials
	rm -rf clients/web/test/ui/target
	rm -rf htmlReports
	rm -rf node_modules

lint: node_modules
	jshint .

.PHONY: all vanadium-binaries go-deps gen-vdl
.PHONY: build-shell build-web-assets build-web serve-web
.PHONY: test test-shell test-web clean lint

.NOTPARALLEL: test
