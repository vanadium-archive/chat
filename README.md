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

## Deployment

TODO(nlacasse): Update this with the new deployment setup once we have one.

The chat application consists of a set of static files hosted on the "v-www" GCE
instance.  Deployment is as simple as pushing to a remote branch on the
instance.  A git post-receive hook on the instance then builds the assets and
makes them available over http.

### First-time setup

You must add your public key to the git user's "authorized_keys".  This will
allow you to push to the git repo on that machine.

From the Developers Console on GCE, ssh into the v-www instance and run:

    sudo su git
    cat >> ~/.ssh/authorized_keys

Paste your ssh public key (commonly in ~/.ssh/id_rsa.pub), and then hit Enter
followed by Ctrl-D.

You may now log out from the instance.

Next, from your local git repository, add a new remote reference to the staging
GCE instance.

    git remote add staging git@staging.v.io:chat.git

### Deploy

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
