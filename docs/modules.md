# Auth and Ledger modules

The root embeds Auth and Ledger command constructors. Each constructor receives
`command.Runtime`, which supplies an authenticated client for a service name.
Modules own their request contracts and command flags. They do not read profiles
or implement Cloud authentication. This boundary supports future packaging
changes; external plugin loading is not implemented. Connectivity is deferred.

## Ledger

The HTTP contract is taken from `formancehq/ledger`, branch `release/v3.0`,
commit `71b0feb549a7cbc22bddaca2859ae5d91a24d378`. Business requests use `/v3`;
server information uses `/_info`. This revision does not contain a public
`pkg/client` module. The CLI therefore uses a small HTTP adapter with route and
payload tests rather than importing server packages or the legacy v2 SDK.
Replace this adapter with the published module client when it becomes available.

```bash
fctl ledger create books
fctl ledger list --page-size 20
fctl ledger show books
fctl ledger --ledger books accounts list
fctl ledger --ledger books accounts show users:001
fctl ledger --ledger books transactions create \
  --data @transaction.json --idempotency-key payment-42
fctl ledger --ledger books transactions list
fctl ledger --ledger books transactions revert 1 --idempotency-key revert-42
```

Request bodies accept inline JSON, `@file`, or `-` for stdin. Use the v3 JSON
schema for postings and Numscript transactions. Amounts remain exact JSON
integers; no floating-point conversion is performed. For example:

```json
{"postings":[{"source":"world","destination":"users:001","asset":"USD/2","amount":1000}]}
```

Lists return one complete server envelope. To continue, pass the server's opaque
cursor to `--cursor` with the same filters and order. Use `--filter` for the v3
filter syntax and `--consistency linearizable|stale` for supported reads.

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

Auth imports `github.com/formancehq/auth/pkg/client` at
`v0.0.0-20251106135031-5373fa4eaeba`. The generated client receives the same
authenticated HTTP client as Ledger; its built-in credential hook is not used.

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
