# Vanadium Chat

Thanks for dogfooding Vanadium Chat!

We currently have two clients: a [web client][client-web] and a [shell
client][client-shell].

Please file bugs and feature requests in our [issue tracker][issue-tracker].

<a name="client-web"></a>
## Running the web client

1. Install the Vanadium Chrome Extension:
  <https://chrome.google.com/webstore/detail/vanadium-extension/jcaelnibllfoobpedofhlaobfcoknpap>

2. Visit the chat webpage: <https://staging.v.io/chat>

<a name="client-shell"></a>
## Running the shell client

These instructions assume you have an up-to-date [Vanadium development
environment] inside `$VANADIUM_ROOT`.

In order to install the shell client, please do the following:

1. Download the chat binary for your architecture:

   * [Linux 64-bit][download-linux-amd64]
   * [OS X 64-bit][download-darwin-amd64]

   (Note, curl won't work because the HTTP request must include your OAuth
   cookies.)

2. Start the Vanadium Security Agent

      $VANADIUM_ROOT/release/go/src/v.io/core/veyron/tools/mgmt/vbash

  You will be prompted for a password, and may have to select blessing caveats
  in your web browser.

  TODO(nlacasse): Is there a better way to get the agent that does not require
  $VANADIUM_ROOT ?

3. Run the chat binary (and use the proxy).

       chmod +x ~/Downloads/chat
       ~/Downloads/chat --veyron.proxy=proxy.envyor.com:8100

 <a name="developing"></a>
## Developing Vanadium Chat

TODO(nlacasse): Fix the tools in tools/ directory so that it's easy to develop
against local services again, and then write the docs here.

[client-shell]: #client-shell
[client-web]: #client-web
[download-linux-amd64]: binaries/chat-linux-amd64/chat
[download-darwin-amd64]: binaries/chat-darwin-amd64/chat
[issue-tracker]: https://github.com/veyron/chat/issues
[Vanadium development environment]: http://go/veyron-development
