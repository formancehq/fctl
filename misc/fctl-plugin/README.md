# fctl legacy service plugin

This library and plugin executable adapt the historical fctl service commands
from main commit `e00243b3e2e56aae6a09d7010b0c17890134388c` to the public fctl SDK.
The repository profile is library/CLI plugin: the fctl host owns authentication,
configuration, terminal interaction and presentation. The plugin has no service
lifecycle, Fx container or independent authentication layer.

`New(*http.Client)` returns the complete `legacy` bundle. `NewService(service,
*http.Client)` returns a service facade using the same SDK contract. Metadata
works with a nil client and no network. `Compatibilities` declares the supported
API families; execution rejects unknown or modern target versions.

## Source layout

```text
misc/fctl-plugin/
  cmd/fctl-plugin-legacy/   standalone transport entry point
  legacy.go                public bundle and service factory assembly
  service.go               service facade and execution version guard
  compatibility.go         supported API families
  internal/
    auth/                  clients, secrets and users
    ledger/                ledgers, accounts, transactions, schemas and logs
    payments/              connectors, accounts, pools and payment operations
    orchestration/         workflows, instances and triggers
    reconciliation/        policies, rules, evaluations and alerts
    wallets/               wallets, balances, holds and movements
    webhooks/              configs and deliveries
    command/               declarative command and HTTP helpers
    metadata/              bundle revision and historical source commit
    testutil/              shared HTTP contract fixtures
  docs/                    command inventory and manual sandbox recipes
```

Each service package owns its manifest, routes, payload validation, aliases and
contract tests. Add an operation to its domain file and register it in that
service's `plugin.go`. `internal/command` supplies common SDK command builders,
route binding and JSON helpers; it does not choose services or contain service
rules. For example, Wallet name resolution belongs to `internal/wallets`, and
workflow polling belongs to `internal/orchestration`.

The module root is the public Go API. Callers use `New`, `NewService` or
`Factories`; implementation packages are internal to this module. Services
import the public SDK and common internal helpers. They do not import one
another, the module root or the fctl core.

## Validation and distribution

```sh
go test -race ./...
go build ./cmd/fctl-plugin-legacy
```

The root fctl Justfile and GoReleaser include this module. The executable calls
`transport.Serve(New)` and obtains HTTP through the host callback; it receives
no tokens in arguments or environment. Its own version is independent of any
historical service release. See the repository's `docs/legacy-plugin.md` for
selection, cache, supported API families and validation.

See [the historical command inventory](docs/port-inventory.md) and
[synthetic sandbox recipes](docs/live-fixtures.md) for the adapted service contracts.
