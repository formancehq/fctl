# Auth and Ledger modules

The root registers embedded Cloud and loads Auth and Ledger commands from
external plugins. The local Ledger and Connectivity adapters have been removed.
Auth has official discovery; Ledger has loader and management support but no
official release in the catalogue. Connectivity commands are absent pending
independent distribution and later integration. Host connection settings remain
available; the migration is not complete.
This page describes the service modules; [Cloud management](cloud.md) describes
control-plane commands. Each plugin implements the public `pluginsdk.Plugin` manifest/execution contract. The core builds Cobra
commands from manifests and supplies an authenticated HTTP client and endpoint.
Plugins own service routes, payloads and command descriptions. They import no
Cobra or core internal packages and do not read profiles or implement Cloud
authentication. See the [plugin contract](plugins.md) and
[plugin distribution](plugin-distribution.md) for discovery, installation and
exact-version selection. See [Connectivity](connectivity.md) for its API and
commands.

## Ledger

Ledger commands require a trusted exact-version external executable installed
for the selected target. A future Ledger product release must publish the
executable and promote its catalogue entries before official sync is available.
The examples below describe the previously validated HTTP command contract;
they do not advertise commands bundled with fctl. Consult the installed product
manifest for its current command tree.

### Historical HTTP contract

The former embedded adapter's HTTP contract was taken from `formancehq/ledger`,
branch `release/v3.0`,
commit `0f4656d1efbcac42705839daccb34012e70d783a`. Business requests use `/v3`;
server information uses `/_info`. This revision does not contain a public
`pkg/client` module. The former adapter used a small HTTP client with route and
payload tests
rather than importing server packages or the legacy v2 SDK. That local adapter
is removed; external command implementations belong to the Ledger product.

```bash
fctl ledger create books
fctl ledger list
fctl ledger show books
fctl ledger --ledger books accounts list --page-size 20
fctl ledger --ledger books accounts list --page-size 20 --after users:001
fctl ledger --ledger books accounts show users:001
fctl ledger --ledger books transactions create \
  --data @transaction.json --idempotency-key payment-42
fctl ledger --ledger books transactions list
fctl ledger --ledger books transactions list --after 42
fctl ledger --ledger books logs list --after 42
fctl ledger --ledger books transactions revert 1 --idempotency-key revert-42
```

Request bodies accept inline JSON, `@file`, or `-` for stdin. Use the v3 JSON
schema for postings and Numscript transactions. Amounts remain exact JSON
integers; no floating-point conversion is performed. For example:

```json
{"postings":[{"source":"world","destination":"users:001","asset":"USD/2","amount":1000}]}
```

JSON output preserves the complete server envelope; terminal tables summarize
lists and display their pagination fields. Use `-o json` for complete payloads.
`ledger list` returns all ledgers and has no pagination or reverse flags.
Accounts, transactions and logs return one page. Continue with `--after` using
the last returned account address or unsigned transaction/log ID; it is an
exclusive boundary sent as the backend's `after` parameter. Keep the same filters
and order. Accounts and transactions support `--reverse`; logs do not.

Only `ledger indexes inspect` uses opaque `--cursor` tokens, taken from its
`nextCursor` response. Regular pages use `--page-size` (default 100; 0 means 100,
with a server cap of 1000). Index inspection accepts page sizes from 1 to 10000.
Use `--filter` for the v3 filter syntax and
`--consistency linearizable|stale` for supported reads. Continuation examples
above illustrate the boundary value; use the last address or ID from your own
previous response.

Other commands cover ledger/account/transaction metadata, balances, stats,
logs, indexes and bulk operations. Use `--help` on each group for its exact
options. Metadata reads return the resource envelope because v3 has no separate
GET metadata endpoint. Log reads require the corresponding index. Bulk results
must be inspected per element; business failures return a nonzero exit code.
There is no v2 log import/export or backup restore compatibility command.

Deletes require `--confirm`. Idempotency keys are supplied by the caller and
never generated automatically. Reuse the same key only for the same logical
operation; inspect server errors before rerunning a mutation.

## Auth

Auth commands belong to the independent
`github.com/formancehq/auth/misc/fctl-plugin` product module. That module imports
`github.com/formancehq/auth/pkg/client` at
`v0.0.0-20251106135031-5373fa4eaeba`. The Auth product plugin module is
not a compile-time dependency of fctl. The external executable uses the host HTTP broker for
authentication; its generated client does not obtain credentials itself.

Auth has no embedded provider. Before executing an Auth command, fctl uses an
installed exact-version executable or discovers and installs a matching release
from the official catalogue. An unavailable official catalogue or missing exact
release produces an explicit error; it does not enable an embedded fallback.
A trusted local build can be installed instead. See the
[local build and official sync examples](plugin-distribution.md#auth-installation).

On a fresh target, `fctl auth --help` performs no network access and displays a
placeholder with sync/install guidance. Once that target is prepared, its full
Auth help and completion use cached command and form metadata offline. The
following commands assume a matching plugin has been installed or is available
for automatic discovery:

```bash
fctl auth info
fctl auth discovery
fctl auth clients list
fctl auth clients create --data '{"name":"automation","scopes":["ledger:read"]}'
fctl auth clients show CLIENT_ID
fctl auth clients update CLIENT_ID --confirm --data '{"description":"Read-only automation"}'
fctl auth clients secrets create CLIENT_ID --data '{"name":"ci"}'
fctl auth clients secrets list CLIENT_ID
fctl auth users list
fctl auth users show USER_ID
```

Updates read the current client and preserve omitted options. Explicit empty
arrays/objects or `false` clear the corresponding settings. Client and secret
deletions require `--confirm`. A newly created secret is deliberately written
to stdout once so the caller can save it. List/show commands expose secret
metadata. Auth has no user CRUD or list pagination in this client contract.

Auth administration and Cloud login are separate operations. Auth's server
does not implement the Membership device-code flow; Cloud login targets
Membership and exchanges its assertion with Auth afterward.
