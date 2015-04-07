# Vanadium Chat

Vanadium Chat is a peer-to-peer chat application that demonstrates common usage
patterns for the [Vanadium][vanadium-home] libraries.

There are currently two clients: a [web client][client-web] and a [shell
client][client-shell].

Please file bugs and feature requests in our [issue tracker][issue-tracker].

## Running a chat client

<a name="client-web"></a>
### Running the web client

1. Install the [Vanadium Chrome Extension][vanadium-extension].

2. Visit the chat webpage: <https://staging.chat.v.io/>

<a name="client-shell"></a>
### Running the shell client

These instructions assume you have an up-to-date [Vanadium development
environment][vanadium-installation] inside `$V23_ROOT`.

In order to install the shell client, please do the following:

1. Build the chat binary.

        cd $V23_ROOT/release/projects/chat
        make build-shell

2. Start the Vanadium Security Agent

        $V23_ROOT/release/go/src/v.io/x/ref/cmd/vbash

  You may be prompted for a password, and may have to select blessing caveats
  in your web browser.

  TODO(nlacasse): Is there a better way to get the agent that does not require
  $V23_ROOT ?

3. Run the chat binary.

        ./clients/shell/go/bin/chat


<a name="architecture"></a>
## Chat architecture

Vanadium Chat is a peer-to-peer chat application.  That means that clients
connect directly to each-other when sending messages.

A side effect of its peer-to-peer nature is that messages are not guaranteed to
arrive in the same order at all clients.

There is currently no storage of chat history.  Aside from tracking the list of
chat room peers, the application is stateless.

The chat application relies on servers for three things:

1. Chat clients mount themselves on a [mounttable server][mounttable], so that
   peers can find them.

2. All RPCs are routed through a [Vanadium proxy][proxy], which allows
   connections to peers who might be behing a NAT.

3. A web server is used to serve web assets to chat clients.

### Client discoverability

When a client joins the chat room, it generates a random string and attempts to
mount itself in the public [mounttable server][mounttable] under the name
`apps/chat/public/<random_string>`.  If that name is already taken, the client
will pick a new random string and try again.

The client sets permissions on that name which prevent peers from mounting on
the same name, but allow them to resolve that name to the client's endpoint.
The client essentially "owns" that particular name in the mounttable.

To find other peers, the client sends a `Glob` RPC to the mounttable, asking
for all names matching `apps/chat/public/*`.  The results of this Glob
correspond to peers in the chat room.  Each result contains the peer's
endpoint and its [blessing pattern][blessings].  The blessing pattern is used
to identify the peer in the chat UI, and to ensure that messages are only sent
to intended recipients.

Chat currently only supports a single chat room called `public`, but eventually
it will support multiple rooms, including private rooms visible to only certain
clients.

### Sending messages

Each chat client implements a `Chat` interface defined in the [VDL][vdl] file
`clients/shell/src/chat/vdl/chat.vdl`.

This interface has a single method, called `SendMessage`, which takes a string
argument (the message) and returns an error, which can be nil.


    type Chat interface {
      // SendMessage sends a message to a user.
      SendMessage(text string) error {}
    }


When a client Alice wants to send a message to her peer Bob, she finds Bob's
entry in the results of the mounttable `Glob`.  She invokes a Vanadium RPC on
that name, calling the `SendMessage` method with her message.  She passes in
Bob's expected remote blessings to the RPC to ensure that the message will
only be sent to Bob, and not to some other malicious client who might have
somehow mounted under Bob's name.  Any returned error indicates an error in
transmission.


<a name="developing"></a>
## Developing Vanadium Chat

This section describes how to set up a local environment to run the chat
clients, and how to build and test the clients.

### Running mounttabled and proxyd locally

To get the chat environment running locally, you will need to run the
mounttable and proxyd servers with a valid identity.  The following script will
request an identity from the identity service and run mounttabled and proxyd
with that identity.

    ./tools/services.sh

You will be prompted to log in with your Google account in order to get an
identity.  You will also be asked to select a blessing.  Leave the values at
their default setting and click "Bless".

Once the identity is in place, the script runs mounttabled on port 8101, and
proxyd on port 8100.  The proxy mounts itself in the mounttable under the name
"proxy".

### Generating VDL dependencies

The `chat.vdl` file defines the server interface implemented by each chat client.
That file can be found in `clients/shell/src/chat/vdl/chat.vdl`.

Running `make gen-vdl` will build the generated JavaScript and Go files in
`clients/web/js/chat/vdl/index.js` and `clients/shell/src/vdl/chat.vdl.go`
which are included by the web and shell clients, respectively.

### Developing the web client

The web client is a [single page application][spa] written with [React][react].
All code is inside `clients/web`.  The Vanadium-specific code is in
`clients/web/js/channel.js`.

A web server is used to serve the js, html, and css to the browser.  You will
need the [Vanadium extension][vanadium-extension] to run the app.

You can build the web assets and run the web server with the following command:

    NOMINIFY=1 make serve-web

This will run the web server on port 4000.  The NOMINIFY environment variable
prevents the code from being minified, making debugging easier.

Once the web server is running, visit the chat app page, and pass the names of
the mountable and proxyd in query params:

    http://localhost:4000?mounttable=/localhost:8101&proxy=proxy

There is a simple suite of web client tests in `clients/web/test`.  You can run
the web client tests with `make test-web`.

### Developing the shell client

The shell client is written in Go with the [Gocui][gocui] UI library.  All
the chat code is in `clients/shell/src/chat`.  The entry-point is in
`clients/shell/src/chat/main.go`.  The Vanadium-specific code is in
`clients/shell/src/chat/channel.go`.

The chat binary is built into `clients/shell/bin/chat`.  You can build it with:

    make build-shell

In order for the chat client to talk to the mounttable and proxy servers, you
will need to get an identity from the identity server:

    export V23_CREDENTIALS=/tmp/vanadium-credentials
    $V23_ROOT/release/go/bin/principal seekblessings

Then run the binary and pass in the v23.namespace.root and v23.proxy flags.
TODO(nlacasse): Update the flag names when they change.

    ./clients/shell/bin/chat --mounttable=/localhost:8101 --proxy=proxy

There is a simple suite of tests for the shell client in
`clients/shell/src/channel_test.go`.  You can run these tests with `make
test-shell`.

## Deployment

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

[blessings]: TODO(nlacasse)
[client-shell]: #client-shell
[client-web]: #client-web
[deploy]: http://goo.gl/QfD4gl
[gocui]: https://github.com/jroimartin/gocui
[infrastructure]: https://vanadium.googlesource.com/infrastructure/+/master/nginx/README.md
[issue-tracker]: https://github.com/vanadium/chat/issues
[mounttable]: TODO(nlacasse)
[proxy]: TODO(nlacasse)
[react]: http://facebook.github.io/react/
[spa]: http://en.wikipedia.org/wiki/Single-page_application
[vanadium-extension]: https://chrome.google.com/webstore/detail/vanadium-extension/jcaelnibllfoobpedofhlaobfcoknpap
[vanadium-install]: https://staging.v.io/installation/
[vanadium-home]: https://staging.v.io/
[vdl]: TODO(nlacasse)
