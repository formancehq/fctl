# Profiles and authentication

The CLI supports standalone service endpoints and Stack gateway endpoints.
All service commands use the same connection boundary. They do not require a
Cloud profile when accessing a local service. Auth and Ledger examples require
a matching external plugin. Official discovery exists for Auth; Ledger currently
requires a trusted local executable or custom catalogue because its official
product release is still pending. Connection settings alone do not install
commands. Connectivity endpoint settings remain available while its service
commands are absent pending independent distribution and later integration.

## Local services without authentication

```bash
fctl profiles add local --auth-mode none \
  --ledger-url http://localhost:9000 --auth-url http://localhost:8080
fctl profiles use local
fctl ledger list
fctl auth info
```

To use a gateway, replace the individual URLs with `--stack-url URL`.
The CLI appends `/api/ledger`, `/api/auth` or `/api/connectivity`. For direct URLs, include any
deployment prefix but not the service's API version: Ledger appends `/v3`.
A configured service URL takes precedence over the gateway URL for that service.
See [Connectivity](connectivity.md) for preserved standalone settings and the
historical command contract.

One-off commands can pass endpoint and authentication flags directly:

```bash
fctl --auth-mode none --ledger-url http://localhost:9000 ledger list
```

## OAuth2 client credentials

```bash
fctl profiles add staging --auth-mode client-credentials \
  --ledger-url https://ledger.example.com \
  --auth-url https://auth.example.com \
  --token-url https://auth.example.com/oauth/token \
  --client-id automation --scopes 'ledger:read auth:read'
# Supply the client secret through your shell or secret manager.
fctl --profile staging ledger list
```

Set `FCTL_CLIENT_SECRET` for the command. The CLI has no client-secret flag and
never saves this secret in a profile. It obtains and renews access tokens
through the configured token endpoint. Select scopes supported by the target
services; writes require corresponding permissions. Token response bodies are
not included in authentication errors. Authenticated service and token endpoints
require HTTPS; plaintext HTTP is allowed only on loopback for local development.

## Cloud user login

```bash
fctl login
fctl cloud organizations list
fctl cloud stack list --organization ORGANIZATION_ID
fctl ledger list --organization ORGANIZATION_ID --stack STACK_ID
fctl auth clients list --organization ORGANIZATION_ID --stack STACK_ID
fctl logout
```

`login` uses `https://app.formance.cloud/api` and creates a profile named
`cloud` when no Cloud profile is selected. An active Cloud profile is
reused; an active local profile is preserved. Explicitly selecting a local
profile for login is an error. Successful login selects the Cloud profile.
A failed or canceled login leaves saved settings, tokens and selection unchanged.

Login opens the default browser and also prints a verification URL and code on
stderr. Use `--no-browser` to display instructions without opening a browser.
If the browser cannot be opened, the printed instructions remain usable.
Results stay on stdout; their format follows [output settings](output.md).
Login verifies the signed Membership identity and
saves its available accesses without requiring an organization or stack.

Membership refresh responses may omit an ID token. When the saved ID token has
expired, its verified signature still binds the historical login subject. A
current verified access JWT and authenticated UserInfo from the same issuer
establish current permissions; all three subjects must match. UserInfo claims
are fetched again rather than persisted as unsigned identity claims.

Service commands resolve a target from explicit flags, saved profile defaults,
then a unique available match in signed Membership claims or current trusted
UserInfo. Ambiguous or missing targets produce a short error with the selection
flags and commands for listing organizations and stacks;
the CLI does not choose an arbitrary target. The first use of a stack may open
the browser again for stack-scoped authorization. Only that scoped Membership
token is exchanged with the stack's Auth service. Per-target tokens are cached
and renewed independently; the root identity token is never sent to the stack.

Current stack grants can contain an ordinary ID token without organization
access claims. The CLI verifies the resource-bound access JWT's signature,
audience (`STACK_URL/api/auth`), organization, stack and subject, and requires
valid granted read/write scopes. The scoped subject must match the root identity.
Historical providers with signed organization/stack access claims remain
supported; their target URL and permissions must match the selected target.

To save target defaults while logging in, use:

```bash
fctl login --organization ORGANIZATION_ID --stack STACK_ID
# Another Membership environment, with its own named profile:
fctl login --profile staging --issuer https://app.staging.formance.cloud/api
```

Target flags on service commands override defaults for that command and do not
change saved settings. `--issuer` and `--client-id` select an identity at login;
changing them for a service command requires logging in again. Direct endpoint
flags cannot override Cloud routes. `logout` removes the root identity and all
cached stack, organization and application tokens locally; it does not revoke
sessions at the provider.

Cloud administration uses Membership and organization grants. Ledger/Auth/Connectivity and
hosted stack helpers use a stack-scoped grant and the gateway. Experimental
`cloud apps` obtains a separate application grant and signed backend audience;
it does not reuse the stack Auth token. See [Cloud management](cloud.md).

## Profiles and overrides

`fctl profiles add NAME` saves a new profile; `--replace` replaces settings and
clears any Cloud login. `fctl profiles use NAME` selects the default profile.
`fctl profiles list` and `fctl profiles show` expose settings without tokens.
Deletion requires `fctl profiles delete NAME --confirm`.

Settings use this precedence: explicit flag, `FCTL_*` environment variable,
saved profile. Examples include `FCTL_LEDGER_URL`, `FCTL_AUTH_URL`, `FCTL_CONNECTIVITY_URL`,
`FCTL_STACK_URL`, `FCTL_AUTH_MODE`, `FCTL_TOKEN_URL` and `FCTL_CLIENT_ID`.
Use `--profile` (`-p`) or `FCTL_PROFILE` to select a saved profile without
changing the default.

```bash
fctl -p staging ledger list
FCTL_PROFILE=staging fctl ledger list
```

The v4 profile store remains `formance/fctl/v4/connections.json` under the
operating system's user configuration directory (`$XDG_CONFIG_HOME` or `~/.config` on Linux,
`~/Library/Application Support` on macOS). Override it with `--config-dir` or
`FCTL_CONFIG_DIR`. The filename is historical: the public name `profiles` does
not change the storage path, JSON format (including the `connections` key), or
locking scheme. Existing v4 profiles and saved sessions are retained without
migration. Cloud tokens are secrets: the file is created with mode 0600 and
updates use atomic replacement. V3 profiles are still not migrated automatically;
the v3 profile directory is not read or modified. Do not put this file in source
control.

Updates are serialized with the existing portable `connections.lock` file lock.
Each session save checks the profile revision, so an in-flight login or refresh
cannot restore tokens after logout, replacement or deletion. If concurrent
commands conflict, retry with the current profile settings.

Cloud token renewal is serialized per profile. Each command reloads the
current session before renewal, so concurrent commands do not consume the same
rotating refresh token. If cancellation follows a successful token rotation,
the CLI allows up to five seconds to save the renewed credentials locally.

HTTP requests use a 30-second timeout, configurable with `--timeout`. Redirects
are refused. The CLI does not automatically retry service writes. A canceled
command cancels its HTTP and authentication work. Results use stdout; errors
and login instructions use stderr. Output defaults to tables on terminals and
JSON elsewhere. Failures return a nonzero exit code.

Use `-d` or `--debug` to trace HTTP requests and responses on stderr. JSON and
form bodies are bounded and credentials are redacted, including authorization
headers, tokens and device codes. Other body formats are summarized.
Results remain on stdout. See [diagnostic redaction](output.md#http-diagnostics)
for the preview limit and retained OAuth error codes.
