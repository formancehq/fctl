# Historical service command inventory

Source baseline: `origin/main` pinned at `e00243b3e2e56aae6a09d7010b0c17890134388c`. All five historical `cmd/<module>` trees were inspected, including helper/controllers and aliases. Generated routes, request JSON names, enums, pagination, and responses were checked against cached `formance-sdk-go/v4@v4.0.0`, the version pinned by that source. The shipped code imports only the Go standard library and public `pluginsdk` / `pluginsdk/httpclient`.

The public module root assembles the five services documented here, plus Auth and Ledger. Their implementations live in separate `internal/<service>` packages. Each manifest uses bundle version `1.0.0` and a stack target. The host supplies HTTP clients, endpoints, authentication and YAML/file/stdin/HTTPS reads. The public legacy service facade applies compatibility guards before execution. These modules perform no file IO and depend on no Cobra, core, UI or auth packages.

## Command surface

There are **111 runnable canonical commands**: Payments 45, Orchestration 16, Reconciliation 23, Wallets 14, Webhooks 13. Every historical API command in these trees is represented; no unsupported command is stubbed.

Paths below are relative to the host-provided service endpoint. Resource IDs are escaped with `httpclient.Path`. Braces indicate a resource argument or wallet target. `list` flags use `cursor` and `page-size` wherever historically supported. Cursor continuation omits first-page query/body parameters for Payments, Wallets and Reconciliation.

### Orchestration — 16

| Command | Method and relative route | Input details |
| --- | --- | --- |
| `workflows list` | GET `/workflows` | No args |
| `workflows show` | GET `/workflows/{id}` | ID |
| `workflows create` | POST `/workflows` | YAML/JSON source or Body; nonempty object stages |
| `workflows run` | POST `/workflows/{id}/instances` | `variable` JSON/CSV pairs or Body; `wait` query |
| `workflows delete` | DELETE `/workflows/{id}` | Confirmation |
| `instances list` | GET `/instances` | `workflow` -> `workflowID`; `running` |
| `instances show` | GET `/instances/{id}`, GET `/workflows/{workflowID}` | Raw instance + associated workflow |
| `instances describe` | GET `/instances/{id}/history`, GET `/instances/{id}/stages/{index}/history` | Additional histories for send stages |
| `instances send-event` | POST `/instances/{id}/events` | Event argument -> `name` |
| `instances stop` | PUT `/instances/{id}/abort` | Confirmation |
| `triggers list` | GET `/triggers` | `name` filter |
| `triggers show` | GET `/triggers/{id}` | ID |
| `triggers create` | POST `/triggers` | Event/workflow arguments; `name`, `filter`, `vars` |
| `triggers delete` | DELETE `/triggers/{id}` | Confirmation |
| `triggers test` | POST `/v2/triggers/{id}/test` | Event JSON argument or Body |
| `triggers occurrences list` | GET `/triggers/{id}/occurrences` | Trigger ID |

### Payments — 45

Every execution reads `/_info` using public `ServiceVersion`, accepting historical top-level/enveloped info and commit versions. V1 SDK operations retain their unversioned routes on Payments 3; the port does not blindly prefix every operation with `/v3`.

| Group | Commands and routes |
| --- | --- |
| `accounts` (4) | `create` POST `/accounts`; `get` GET `/accounts/{id}`; `list` GET `/accounts`; `balances` GET `/accounts/{id}/balances` |
| `payments` (4) | `create` POST `/payments`; `get` GET `/payments/{id}`; `list` GET `/payments`; `set-metadata` PATCH `/payments/{id}/metadata` with flat string metadata object |
| `bank-accounts` (5) | `create` POST, `list` GET `/bank-accounts`; `get` GET `/bank-accounts/{id}`; `forward` POST `/bank-accounts/{id}/forward` with `connectorID`; `update-metadata` PATCH `/bank-accounts/{id}/metadata` with metadata envelope; all select `/v3` on Payments 3 |
| `pools` (9) | `create` POST `/pools` (V3 on 3); `get` GET `/pools/{id}`; `list` GET `/pools`; `delete` DELETE `/pools/{id}`; `latest-balances` GET `/pools/{id}/balances/latest` (V3 on 3); `balances` GET `/pools/{id}/balances?at=...`; `add-account` POST `/pools/{id}/accounts` with `accountID`; `remove-account` DELETE `/pools/{id}/accounts/{accountID}`; `update-query` PATCH `/v3/pools/{id}/query` (>=3.1) |
| `transfer-initiation` (9) | `create` POST `/transfer-initiations`; `get` GET `/transfer-initiations/{id}`; `list` GET `/transfer-initiations`; `delete` DELETE `/transfer-initiations/{id}`; `retry` POST `/transfer-initiations/{id}/retry`; `reverse` POST `/transfer-initiations/{id}/reverse`; `update-status` POST `/transfer-initiations/{id}/status` (unavailable on 3); `approve` / `reject` POST `/v3/payment-initiations/{id}/approve` or `/reject` (3 required) |
| `connectors` (9) | `list`, `list-available`, `install`, `get-config`, `update-config`, `uninstall`; `schedules list`, `schedules get`, `schedules instances list` |
| `orders` (2) | `list` GET `/v3/orders`; `get` GET `/v3/orders/{id}` (>=3.3) |
| `conversions` (2) | `list` GET `/v3/conversions`; `get` GET `/v3/conversions/{id}` (>=3.3) |
| `tasks` (1) | `get` GET `/v3/tasks/{id}` (3 required) |

Connector V3 routes: GET `/v3/connectors`, GET `/v3/connectors/configs`, POST `/v3/connectors/install/{provider}`, GET/PATCH `/v3/connectors/{id}/config`, DELETE `/v3/connectors/{id}`. Install/update resolve provider casing via configs, override its payload field without losing numeric precision, and retain the historical requested-name fallback for unlisted development providers. Backend errors propagate. Schedules use `/v3/connectors/{id}/schedules`, `/{scheduleID}`, `/{scheduleID}/instances`.

Connector legacy routes: POST `/connectors/{PROVIDER}`; POST `/connectors/{PROVIDER}/{id}/config`; GET/DELETE provider/ID routes, with the provider-only v0 config/uninstall variant; GET `/connectors/configs` and GET `/connectors`. `provider` and `connector-id` select targets; get-config discovers a unique connector when legacy inputs are incomplete. Legacy list exposes `id` alongside original `connectorID` for declarative choices. Typed legacy provider names are mapped explicitly; V3 remains dynamic.

Orders/conversions filters use a GET JSON `$and` of `$match` clauses with snake_case field names: `connector-id`, `reference`, `status`, `source-asset`, `destination-asset`; orders also expose `direction` and `type`. Account/payment amounts retain arbitrary precision; required request fields, timestamps, enum values, metadata values and pool version requirements are validated explicitly. All historical confirmed operations retain confirmations; all deletes require them.

### Reconciliation — 23

| Commands | Method and relative route | Input details |
| --- | --- | --- |
| `list`, `get` | GET `/reconciliations`, `/reconciliations/{id}` | Pagination / ID |
| `policies list`, `policies get` | GET `/policies`, `/policies/{id}` | Pagination / ID |
| `policies create`, `policies delete` | POST `/policies`, DELETE `/policies/{id}` | Required name, ledgerName, paymentsPoolID, ledgerQuery; confirmations |
| `policies reconcile` | POST `/policies/{id}/reconciliation` | RFC3339 ledger/payments timestamps |
| `rules list`, `rules get` | GET `/rules`, `/rules/{id}` | Query-builder JSON GET body on first page / ID |
| `rules create`, `rules update`, `rules delete` | POST `/rules`, PATCH/DELETE `/rules/{id}` | Object JSON create/patch preserves unknown template fields and explicit false; confirmations |
| `rules evaluate` | POST `/rules/{id}/evaluate` | `at`, nonnegative `safety-margin`, per-source `source-pit`; confirmation |
| `evaluations list`, `evaluations get` | GET `/evaluations`, `/evaluations/{id}` | Query-builder JSON / ID |
| `alerts list`, `alerts get`, `alerts events` | GET `/alerts`, `/alerts/{id}`, `/alerts/{id}/events` | Query-builder JSON / ID / pagination |
| `alerts ack`, `resolve`, `accept`, `snooze`, `unsnooze` | POST `/alerts/{id}/{transition}` | Required `by`; `note` required for accept; `transaction-ref` for resolve; future `until` for snooze; confirmations |

Clarity template payloads stay open object schemas, matching the historical raw API client. Backend validation determines supported template kinds/specs. There is no fabricated delete for an evaluation or alert.

### Wallets — 14

| Commands | Method and relative route | Input details |
| --- | --- | --- |
| `create`, `update`, `list`, `show` | POST `/wallets`, PATCH `/wallets/{id}`, GET `/wallets`, GET `/wallets/{id}` | Name, metadata, `ik`; list pagination/name/metadata; show `id` or `name` |
| `credit`, `debit` | POST `/wallets/{id}/credit` or `/debit` | Arbitrary precision amount/asset; metadata, balance(s), `ik`; credit sources; debit destination/pending/description |
| `balances create`, `list`, `show` | POST/GET `/wallets/{id}/balances`, GET `/wallets/{id}/balances/{name}` | Expiry, integer priority; pagination; balance name |
| `holds list`, `show`, `confirm`, `void` | GET `/holds`, `/holds/{id}`; POST `/holds/{id}/confirm` or `/void` | Wallet/metadata filters; confirm amount/final/ik; void ik |
| `transactions list` | GET `/transactions` | Wallet filter, pagination |

Wallet selection accepts `id` or exact `name`, resolves names through a bounded read and rejects zero/multiple matches. Subject forms are `account=ADDRESS`, `wallet=id:ID[/BALANCE]`, `wallet=name:NAME[/BALANCE]`. Metadata filters use deep-object keys. Writes retain their confirmation gates. **Wallet deletion is absent from the historical CLI and API and remains absent.**

### Webhooks — 13

| Commands | Method and relative route | Input details |
| --- | --- | --- |
| `list`, `create`, `update`, `delete` | GET/POST `/configs`, PUT/DELETE `/configs/{id}` | List ID/endpoint; config endpoint/events/optional 24-byte base64 signing secret; confirmations for writes |
| `change-secret`, `activate`, `deactivate`, `test` | PUT `/configs/{id}/secret/change`, `/activate`, `/deactivate`; GET `/configs/{id}/test` | Secret optional; all confirm because test sends an event |
| `deliveries list`, `show`, `attempts` | GET `/deliveries`, `/deliveries/{id}`, `/deliveries/{id}/attempts` | Pagination, config/status/date filters |
| `deliveries replay`, `replay-bulk` | POST `/deliveries/{id}/replay`, `/deliveries/replay` | Required idempotency key and confirmation; bulk start/end, failed/pending status list, config IDs, cursor/page size |

Update distinguishes omitted signing secret from explicitly empty secret. Replay validates its status/time bounds and preserves request idempotency headers.

## File argument compatibility — 13 commands

The generic payload helper declares `Files.ReadArgument` at the preexisting maximum positional index, sets YAML decoding (also accepts JSON), increments Args.Max and appends `[<file>|-]` to Use. Host-provided Body/form/`--data` alternatives remain available. The module callback consumes Body and performs no file access.

- Orchestration: `workflows create`.
- Payments: `accounts create`, `bank-accounts create`, `payments create`, `pools create`, `pools update-query`, `transfer-initiation create`, `transfer-initiation reverse`, `connectors install`, `connectors update-config`.
- Reconciliation: `policies create`, `rules create`, `rules update`.

These are exactly the `<file>|-` signatures found in the five pinned trees. Query flags, workflow variable bodies, and trigger event JSON arguments are not annotated as historical files.

## Alias compatibility and limitations

Canonical underscore names become `bank-accounts`, `transfer-initiation`, `update-status`. Their historical `bank_accounts`, `transfer_initiation`, `update_status` names remain declared aliases. The additional `payment_initiations` compatibility alias also resolves to `transfer-initiation`. Other unambiguous historical aliases are retained. Conflicting `p` belongs to payments, `cr` to wallet create, `r` to transfer retry. The historical `holds h` command alias is retained; the distinct `-h` flag continues to show help.

No historical API commands are outstanding. Unsupported backend operations return API errors; guards explain version-specific commands. File IO and CLI rendering moved to the host. Old pterm views are replaced by raw JSON responses. The accidentally accepted, unused third wallet-debit argument is rejected rather than silently ignored.

## Verification

From the repository root, run `nix develop --impure --command just pre-commit`
and `nix develop --impure --command just tests`. These include the core, public
SDK and independent legacy module. From `misc/fctl-plugin`, run
`go test -race ./internal/...` to check service implementations and shared tools.

HTTP contract tests cover reads/writes/deletes, path escaping, exact
query/body/header names, version/provider routing, precision above 2^53, API
errors/partial responses, dependent reads before writes, invalid input without
HTTP mutation, cancellation, manifest independence, underscore aliases and
every historical file-source signature.

See [synthetic sandbox recipes](live-fixtures.md) for manual examples. The
repository's `docs/legacy-validation.md` records verified Cloud results and
remaining runtime limits.
