# Connections and authentication

The CLI supports standalone service endpoints and Stack gateway endpoints.
All service commands use the same connection boundary. They do not require a
Cloud profile when accessing a local service.

## Local services without authentication

```bash
fctl connections add local --auth-mode none \
  --ledger-url http://localhost:9000 --auth-url http://localhost:8080
fctl connections use local
fctl ledger list
fctl auth info
```

To use a gateway, replace the individual URLs with `--stack-url URL`.
The CLI appends `/api/ledger` or `/api/auth`. For direct URLs, include any
deployment prefix but not the service's API version: Ledger appends `/v3`.
A configured service URL takes precedence over the gateway URL for that service.

One-off commands can pass connection flags directly:

```bash
fctl --auth-mode none --ledger-url http://localhost:9000 ledger list
```

## OAuth2 client credentials

```bash
fctl connections add staging --auth-mode client-credentials \
  --ledger-url https://ledger.example.com \
  --auth-url https://auth.example.com \
  --token-url https://auth.example.com/oauth/token \
  --client-id automation --scopes 'ledger:read auth:read'
# Supply the client secret through your shell or secret manager.
fctl --connection staging ledger list
```

Set `FCTL_CLIENT_SECRET` for the command. The CLI has no client-secret flag and
never saves this secret in a connection. It obtains and renews access tokens
through the configured token endpoint. Select scopes supported by the target
services; writes require corresponding permissions. Token response bodies are
not included in authentication errors. Authenticated service and token endpoints
require HTTPS; plaintext HTTP is allowed only on loopback for local development.

## Cloud user login

```bash
fctl login
fctl ledger list --organization ORGANIZATION_ID --stack STACK_ID
fctl auth clients list --organization ORGANIZATION_ID --stack STACK_ID
fctl logout
```

`login` uses `https://app.formance.cloud/api` and creates a connection named
`cloud` when no Cloud connection is selected. An active Cloud connection is
reused; an active local connection is preserved. Explicitly selecting a local
connection for login is an error. Successful login selects the Cloud connection.
A failed or canceled login leaves saved settings, tokens and selection unchanged.

Login opens the default browser and also prints a verification URL and code on
stderr. Use `--no-browser` to display instructions without opening a browser.
If the browser cannot be opened, the printed instructions remain usable.
JSON results stay on stdout. Login verifies the signed Membership identity and
saves its available accesses without requiring an organization or stack.

Service commands resolve a target from explicit flags, saved connection defaults,
then a unique available match in the verified Membership claims. Ambiguous or
missing targets produce an error listing the available organization/stack IDs;
the CLI does not choose an arbitrary target. The first use of a stack may open
the browser again for stack-scoped authorization. Only that scoped Membership
token is exchanged with the stack's Auth service. Per-target tokens are cached
and renewed independently; the root identity token is never sent to the stack.

To save target defaults while logging in, use:

```bash
fctl login --organization ORGANIZATION_ID --stack STACK_ID
# Another Membership environment, with its own named connection:
fctl login --connection staging --issuer https://app.staging.formance.cloud/api
```

Target flags on service commands override defaults for that command and do not
change saved settings. `--issuer` and `--client-id` select an identity at login;
changing them for a service command requires logging in again. Direct endpoint
flags cannot override Cloud routes. `logout` removes the root identity and all
cached target tokens locally; it does not revoke sessions at the provider.

## Profiles and overrides

`connections add NAME` saves a new profile; `--replace` replaces settings and
clears any Cloud login. `connections list` and `connections show` expose settings
without tokens. Deletion requires `connections delete NAME --confirm`.

Settings use this precedence: explicit flag, `FCTL_*` environment variable,
saved connection. Examples include `FCTL_LEDGER_URL`, `FCTL_AUTH_URL`,
`FCTL_STACK_URL`, `FCTL_AUTH_MODE`, `FCTL_TOKEN_URL` and `FCTL_CLIENT_ID`.
Use `--connection` or `FCTL_CONNECTION` to select a saved connection without
changing the default.

The v4 store is `formance/fctl/v4/connections.json` under the operating system's
user configuration directory (`$XDG_CONFIG_HOME` or `~/.config` on Linux,
`~/Library/Application Support` on macOS). Override it with `--config-dir` or
`FCTL_CONFIG_DIR`. Cloud tokens are secrets: the file is created with mode 0600
and updates use atomic replacement. The v3 profile directory is not read or
modified. Do not put this file in source control.

Updates are serialized with a portable file lock. Each session save checks the
connection revision, so an in-flight login or refresh cannot restore tokens
after logout, replacement or deletion. If concurrent commands conflict, retry
with the current connection settings.

Cloud token renewal is serialized per connection. Each command reloads the
current session before renewal, so concurrent commands do not consume the same
rotating refresh token. If cancellation follows a successful token rotation,
the CLI allows up to five seconds to save the renewed credentials locally.

HTTP requests use a 30-second timeout, configurable with `--timeout`. Redirects
are refused. The CLI does not automatically retry service writes. A canceled
command cancels its HTTP and authentication work. Output is JSON on stdout;
errors and login instructions use stderr. Failures return a nonzero exit code.

Use `-d` or `--debug` to trace HTTP requests and responses on stderr. JSON and
form bodies are bounded and credentials are redacted, including authorization
headers, tokens and device codes. Other body formats are summarized. Service
JSON results remain on stdout.
