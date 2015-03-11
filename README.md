# Vanadium Chat

There are currently have two clients: a [web client][client-web] and a [shell
client][client-shell].

Please file bugs and feature requests in our [issue tracker][issue-tracker].

<a name="client-web"></a>
## Running the web client

1. Install the Vanadium Chrome Extension:
  <https://chrome.google.com/webstore/detail/vanadium-extension/jcaelnibllfoobpedofhlaobfcoknpap>

2. Visit the chat webpage: <https://staging.chat.v.io/>

<a name="client-shell"></a>
## Running the shell client

These instructions assume you have an up-to-date [Vanadium development
environment] inside `$VANADIUM_ROOT`.

In order to install the shell client, please do the following:

1. Build the chat binary.

      cd $VANADIUM_ROOT/release/projects/chat
      make build-shell

2. Start the Vanadium Security Agent

      $VANADIUM_ROOT/release/go/src/v.io/x/ref/cmd/mgmt/vbash

  You may be prompted for a password, and may have to select blessing caveats
  in your web browser.

  TODO(nlacasse): Is there a better way to get the agent that does not require
  $VANADIUM_ROOT ?

3. Run the chat binary.

       $VANADIUM_ROOT/release/projects/chat/clients/shell/go/bin/chat

<a name="developing"></a>
## Developing Vanadium Chat

### Running mounttable, proxyd, and web server locally

To get the chat environment running locally, you will need to run mounttable
and proxyd servers with a valid identity.  The following script will request
and identity from the identity service and run mounttabled and proxyd with that
identity.

      ./tools/services.sh

You will be prompted to log in with your google account in order to get an
identity.  You will also be asked to select a blessing.  Leave the values at
their default setting and click "Bless".

Once the identity is in place, the script runs mounttabled on port 8101, and
proxyd on port 8100.  The proxy mounts itself in the mounttable under the name
"proxy".

### Running the web server and web app in dev mode

You can run the chat web server with:

      NOMINIFY=1 make serve-web

This will run the web server on port 4000.  The NOMINIFY environment varible
prevents the code from being minified, making debugging easier.

Once the web server is running, visit the chap app page, and pass the names of
the mountable and proxyd in query params.

      http://localhost:4000?mounttable=/localhost:8101&proxy=proxy

### Running the chat app in dev mode

Build the chat binary:

      make build-shell

Get an identity from the identity server:

  export VEYRON_CREDENTIALS=/tmp/vanadium-credentials
  $VANADIUM_ROOT/release/go/bin/principal seekblessings

Then run the binary and pass in the veyron.namespace.root and veyron.proxy
flags.

       ./clients/shell/bin/chat --mounttable=/localhost:8101 --proxy=proxy

## Deploy

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

[client-shell]: #client-shell
[client-web]: #client-web
[deploy]: http://goo.gl/QfD4gl
[infrastructure]: https://vanadium.googlesource.com/infrastructure/+/master/nginx/README.md
[issue-tracker]: https://github.com/vanadium/chat/issues
[Vanadium development environment]: https://dev.v.io/installation/
