# Chat client: shell

Shell client for the chat app.

## Building

    make build-shell

## Running with envyor services

    ./tools/client_shell.sh

## Running with local mounttable

Make sure mounttabled and identity tool are compiled:

    $VEYRON_ROOT/scripts/build/go install \
      veyron.io/veyron/veyron/services/mounttable/mounttabled \
      veyron.io/veyron/veyron/tools/principal

Run a mounttable on port 6000:

    mounttabled --address :6000

Seek a blessing:

    VEYRON_CREDENTIALS=/tmp/chat.credentials principal seekblessings

Run the client:

    NAMESPACE_ROOT=/localhost:6000 \
      VEYRON_CREDENTIALS=/tmp/chat.credentials \
      ./bin/client
