# Historical Ledger plugin

This package ports every executable command in `cmd/ledger` from fctl
`origin/main` pinned at `e00243b3e2e56aae6a09d7010b0c17890134388c`.
HTTP contracts were checked against the generated `formance-sdk-go/v4@v4.0.0`
source used by that revision. It uses only the public `pluginsdk` and
`pluginsdk/httpclient` packages and the Go standard library.

`New(*http.Client) pluginsdk.Plugin` exposes an offline manifest with root and
service `ledger`, version `1.0.0` (the legacy bundle version). The parent bundle
owns service-version selection, execution guards and its executable. No server
version guard runs in this package.

`ExecuteRequest.Endpoint` must already be the Ledger service base, including
the gateway prefix where needed, for example `https://stack.example/api/ledger`.
The injected client owns authentication. Relative v1 routes have **no `/v1`**
prefix. v2 routes use `/v2`. No modern Ledger v3 routes are used.

## Command and route inventory

All 23 historical executable commands are included. `{ledger}` below comes
from the inherited `--ledger` flag (default `default`) unless a ledger name is
an explicit argument. Routes are relative to the service endpoint.

| Command | Method and route |
| --- | --- |
| `create NAME` | `POST /v2/NAME` |
| `send [SOURCE] DESTINATION AMOUNT ASSET` | `POST /{ledger}/transactions` |
| `stats` | `GET /{ledger}/stats` |
| `server-infos` | `GET /_info` |
| `list` | `GET /v2` |
| `set-metadata NAME KEY=VALUE...` | `PUT /v2/NAME/metadata` |
| `delete-metadata NAME KEY` | `DELETE /v2/NAME/metadata/KEY` |
| `export` | `POST /v2/{ledger}/logs/export` |
| `import NAME [SOURCE]` | `POST /v2/NAME/logs/import` |
| `accounts list` | `GET /v2/{ledger}/accounts` |
| `accounts show ADDRESS` | `GET /{ledger}/accounts/ADDRESS` |
| `accounts set-metadata ADDRESS KEY=VALUE...` | `POST /{ledger}/accounts/ADDRESS/metadata` |
| `accounts delete-metadata ADDRESS KEY` | `DELETE /v2/{ledger}/accounts/ADDRESS/metadata/KEY` |
| `transactions list` | `GET /{ledger}/transactions` |
| `transactions num [SOURCE]` | `POST /{ledger}/transactions` with a script |
| `transactions revert ID` | `POST /{ledger}/transactions/ID/revert?disableChecks=BOOL` |
| `transactions revert ID --at-effective-date` | `POST /v2/{ledger}/transactions/ID/revert?atEffectiveDate=true&force=BOOL` |
| `transactions show ID` | `GET /{ledger}/transactions/ID` |
| `transactions set-metadata ID KEY=VALUE...` | `POST /{ledger}/transactions/ID/metadata` |
| `transactions delete-metadata ID KEY` | `DELETE /v2/{ledger}/transactions/ID/metadata/KEY` |
| `schemas insert VERSION [SOURCE]` | `POST /v2/{ledger}/schemas/VERSION` |
| `schemas get VERSION` | `GET /v2/{ledger}/schemas/VERSION` |
| `schemas list` | `GET /v2/{ledger}/schemas` |
| `volumes list` | `GET /v2/{ledger}/volumes` |

`send` defaults SOURCE to `world`. Amounts and transaction IDs use arbitrary
precision integers. `last` and `lastN` resolve via the historical v1 transaction
list with `pageSize=1`; `lastN` subtracts N from the final ID, rather than selecting
the Nth row. No automatic mutation retry runs.

Transactions list preserves the historical `pageSize=5`, `account`, `source`,
`destination`, `reference`, `startTime`, `endTime`, and `metadata[KEY]` queries.
Accounts and volumes filters use a JSON `{"$and":[{"$match":...}]}` body on GET,
as in the historical SDK. Ledger list sends its historical JSON `null` filter
and `includeDeleted=false`. Volumes preserves `pageSize=10`, `groupBy=0`, and
`insertionDate=false` defaults. Schemas preserves `pageSize=15`, `order=desc`, and
`sort=created_at`, including the generated SDK's page-size default with a cursor.
Volumes also retains the historical page-size query with a cursor.

## Inputs and host integration

The SDK has string, boolean and uint32 flags. Historical string-slice flags
(`metadata`, `features`, `account-var`, `portion-var`, `amount-var`) are declared
as strings. Supply comma-separated `key=value` pairs or a JSON array of strings:

```text
--metadata 'owner=alice,environment=test'
--metadata '["description=hello, world","reference=a=b"]'
--amount-var 'payment=123456789012345678901234567890/USD/2'
```

Pairs split at the first equals sign; repeated keys keep the last value. Use a
JSON string array for embedded commas or newlines. Flags retain clear help
describing these formats. Explicit `ChangedFlags` allow clearing Numscript body
metadata, reference and timestamp.

Declarative forms include ledger/account/transaction/schema selectors, metadata
arguments, send arguments, Numscript text and schema chart input. List commands
accept `cursor` and `page-size` for selector enumeration. Ledger/account lists
keep the server's historical page size when omitted. Discovery invokes only GET
operations. The host owns interaction, confirmation and file access.

The plugin never opens a file, reads stdin, writes a terminal, fetches a schema
source URL, or reads environment variables. The manifest declares file
metadata; the parent host fills `ExecuteRequest.Body`:

| Command | Historical host source/output | SDK Body/result |
| --- | --- | --- |
| `transactions num` | Filename or `-` argument 0 | Body: JSON string of Numscript, `{ "plain": "...", "vars": {} }`, or `{ "script": { "plain": "...", "vars": {} }, "metadata": {}, "reference": "...", "timestamp": "..." }` |
| `import` | Filename argument 1 or `--file`, or stdin | Body: JSON array of raw log objects or JSON string containing NDJSON |
| `schemas insert` | JSON/YAML file or URL argument 1 | Body: JSON schema object; the host converts YAML |
| `export` | `--file` or stdout | Result: JSON array of raw logs; host renders NDJSON |
| `schemas get` | `--format json/yaml/yml` | Result: raw HTTP JSON; host unwraps/presents schema as needed |

`--data` is the declared body flag for direct JSON input. Source arguments are
optional for direct SDK callers supplying Body. Script bodies retain unknown
JSON fields and exact numbers. Schema insertion preserves its raw JSON body.
HTTP JSON responses retain their original envelope, unknown fields and exact
integers, including nested or direct server-info responses. A 204 returns JSON
`null`. Structured service-error bodies accompany the returned error.

Import sends octet-stream NDJSON batches of 100 logs. All logs are validated
before a request. `--resume-from-last-log` looks up the latest log through
`GET /v2/NAME/logs?pageSize=1` and skips through that exact ID in Body. A missing
checkpoint fails before import writes. Import results report `imported`,
`skipped`, and `batches`; on failure these count only acknowledged successful
batches and include the raw service-error body. Export accepts arbitrary NDJSON
line lengths within its total bound and preserves a final line without newline.

## Compatibility limits and verification

- Historical command aliases are declared in the SDK manifest and resolve
  to canonical paths.
- Environment-derived dates, terminal tables/colors/progress, filesystem
  reads/writes, YAML rendering and source URL fetching belong to the host.
- SDK Body is limited to 4 MiB; export is buffered up to the public HTTP
  boundary's 32 MiB response limit. Very large exports need host/transport
  streaming support. Import may span multiple host invocations with resume.
- Validation rejects empty/dot path identifiers, malformed pairs, negative
  amounts/IDs/lastN offsets and invalid dates before HTTP. This intentionally
  avoids historical cases that could fail later at the backend or panic.
- Schema structure and Numscript input shape are validated locally; full
  schema semantics and program execution remain backend responsibilities.
- Host integration and sandbox live QA are performed by the parent task.

Run the parent module's `go test -race ./...`. Contract tests use `httptest` and
cover methods/routes, query/filter/payload shapes, exact big integers, validation
before HTTP, both revert APIs, last/lastN, raw JSON/error envelopes, cancellation,
import batching/resume/partial results, and export NDJSON conversion.
