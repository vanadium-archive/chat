# Chat app

## Links

- [Design doc](http://go/veyron-chat-app)
- [Veyron Chrome extension](https://github.com/veyron/veyron.js/raw/master/extension/veyron.crx)

## Clients

We provide a web client and a shell client. See their respective README.md files
for usage and development instructions.

## Notes

To glob the local mounttable:

    $VEYRON_ROOT/veyron/go/bin/mounttable glob /:4001 '*'
    $VEYRON_ROOT/veyron/go/bin/mounttable glob /:4001/apps/chat/public '*'

To glob the proxy.envyor.com mounttable:

    $VEYRON_ROOT/veyron/go/bin/mounttable glob /proxy.envyor.com:8101 '*'

## Prerequisites

You will need Go 1.4, Git, Mercurial, and `make` to build the shell client and
dependencies.

In addition, you will need Node.js and npm to build the web client, and Chrome
to run the client and the tests.

# Deploy

If you do not have access to the vanadium-staging GCE account ping
jasoncampbell@. Once you have access you will need to login to the account via
the command line.

    gcloud auth login

To deploy the site to https://staging.chat.v.io use the make target
`deploy-staging`.

    make deploy-staging

This will sync the build directory to the private Google Storage bucket
`gs://staging.chat.v.io` which gets automatically updated to the nginx
front-end servers. Currently all static content is protected by OAuth. For
more details on the deployment infrastructure see [this doc][deploy] and the
[infrastructure] repository.

### Old Deploy

TODO(nlacasse): Update this with the new deployment setup

Build and deploy the web assets to staging:

    ./tools/deploy.sh web

Build and deploy the shell client binaries to staging:

    ./tools/deploy.sh shell

Build and deploy everything to staging:

    ./tools/deploy.sh all

### Release

Cutting a release does a deploy, and if the deploy is successful, tags the
current commit with the version and pushes it to the veyron remote.

Make sure you have added the veyron remote.

    git remote add veyron git@github.com:veyron/chat.git

Run the release.sh script with the desired version.  Version must be of the
form "v1.2.3".

    ./tools/release.sh <version>

[deploy]: http://goo.gl/QfD4gl
[infrastructure]: https://vanadium.googlesource.com/infrastructure/+/master/nginx/README.md
