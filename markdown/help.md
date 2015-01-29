# Vanadium Chat

Thanks for dogfooding Vanadium Chat!

We currently have two clients: a [web client][client-web] and a [shell
client][client-shell].

Please file bugs and feature requests in our [issue tracker][issue-tracker].

<a name="client-web"></a>
## Running the web client

These instructions assume you have an up-to-date [Veyron development
environment] inside `$VEYRON_ROOT`.

*NOTE*: You also must be using a Chrome profile that is associated with your
google.com account. If you are using a profile that is not associated with any
account, you will be prompted to log in, and you must do so with your
google.com account. If you are using a profile that is associated with a
non-google.com account, you will be rejected by the identity server.

In order to use the web client, please do the following:

1. Enable insecure WebSockets from https origins.

   * Open a new tab to
     <chrome://flags/#allow-insecure-websocket-from-https-origin>.
   * Click "Enable" to allow insecure WebSockets from https origins.
   * **Restart Chrome** to let the change take effect.

   *Note*: This step is only temporary. Once WSPR is running inside the Veyron
   Chrome extension and communicating via the `postMessage` API, this step
   will go away. Also note that this security policy was only [added
   recently][chromium-cl-insecure-websockets], in Chrome 36. Furthermore,
   Chrome is considering relaxing the policy to allow [insecure WebSocket
   connections to localhost][chromium-issue-remove-insecure-websocket-flag],
   which is our use case.

2. Install the Veyron Chrome extension.

   * Download the [Veyron extension][veyron-chrome-extension].
   * Open a new tab to <chrome://extensions>.
   * Drag-and-drop the Veyron extension into the extensions tab.

   *Note*: Once the Veyron Chrome extension is hosted in the Chrome Web Store,
   this will be a one-click download+install process.

3. Get a blessing for WSPR.

       veyron go install veyron.io/veyron/veyron/tools/principal
       export VEYRON_CREDENTIALS=/tmp/chat.credentials
       $VEYRON_ROOT/veyron/go/bin/principal seekblessings

   This will open a new tab to the identity server. **Set the expiration to
   "730h" (one month)**, or some other large number of hours. Click "Bless",
   wait for a confirmation message, then close the tab.

4. Start WSPR.

       veyron go install veyron.io/wspr/veyron/services/wsprd
       $VEYRON_ROOT/veyron/go/bin/wsprd \
         --identd=/proxy.envyor.com:8101/identity/veyron-test/google \
         --veyron.proxy=proxy.envyor.com:8100 --v=1 --alsologtostderr=true

   *Note*: This step is only temporary. Once WSPR is running inside of the
   Veyron Chrome extension, you will not need to do this.

5. Visit the chat webpage: <https://staging.v.io/chat>

<a name="client-shell"></a>
## Running the shell client

These instructions assume you have an up-to-date [Veyron development
environment] inside `$VEYRON_ROOT`.

In order to install the shell client, please do the following:

1. Download the chat binary for your architecture:

   * [Linux 64-bit][download-linux-amd64]
   * [OS X 64-bit][download-darwin-amd64]

   (Note, curl won't work because the HTTP request must include your OAuth
   cookies.)

2. Get a blessing.

       veyron go install veyron.io/veyron/veyron/tools/principal
       export VEYRON_CREDENTIALS=/tmp/chat.credentials
       $VEYRON_ROOT/veyron/go/bin/principal seekblessings


   This will open a new tab to the identity server. **Set the expiration to
   "730h" (one month)**, or some other large number of hours. Click "Bless",
   wait for a confirmation message, then close the tab.

3. Run the chat binary (and use the proxy).

       chmod +x ~/Downloads/chat
       ~/Downloads/chat --veyron.proxy=proxy.envyor.com:8100

<a name="developing"></a>
## Developing Vanadium Chat

* Make sure your ssh keys have been added to the [Veyron GitHub organization][veyron-github-org].

* Clone the repository.

      git clone git@github.com:veyron/chat.git
      cd chat

* Build and run the required services for the web client.

      ./tools/client_web.sh

* Build and run the required services for the shell client (including the client
  binary itself).

      ./tools/client_shell.sh

* Build and run all services needed to host the chat application, including a
  mounttable, identityd, proxyd, and chat webserver (for serving the webapp
  bundle). Note, this won't work until the security model changes have been
  completed.

      ./tools/host.sh

[allow-insecure-websockets]: chrome://flags/#allow-insecure-websocket-from-https-origin
[chromium-cl-insecure-websockets]: https://codereview.chromium.org/246893014/
[chromium-issue-remove-insecure-websocket-flag]: https://code.google.com/p/chromium/issues/detail?id=367149
[client-shell]: #client-shell
[client-web]: #client-web
[download-linux-amd64]: binaries/chat-linux-amd64/chat
[download-darwin-amd64]: binaries/chat-darwin-amd64/chat
[issue-tracker]: https://github.com/veyron/chat/issues
[Veyron development environment]: http://go/veyron-development
[veyron-chrome-extension]: veyron.crx
[veyron-github-org]: https://github.com/veyron
