# Chat client: web

Web client for the chat app.

## Development

To develop locally, run `make build-web`, then run the following commands in
different terminals:

    tools/host.sh
    node server.js --hostname="0.0.0.0" --port="4000"

Be sure to add "?mtaddr=$(hostname):8101" to the url.

To use proxy.envyor.com services, run `make build-web`, then run these commands
instead:

    tools/client_web.sh
    node server.js --hostname="0.0.0.0" --port="4000"

### Editing veyron.js

To modify veyron.js while working on the chat app, use `npm link`:

    cd ${VEYRON_ROOT}/veyron.js
    npm link
    cd -
    npm link veyron

## TODO

- Protect against various mounttable security issues, e.g. to prevent users from
  impersonating one another. Note, some of these protections require features to
  be added to the mounttable API. Dave's working on it.
