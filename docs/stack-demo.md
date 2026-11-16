# Stack demo generator

`fctl stack demo` continuously runs a built-in financial recipe against a selected
stack. It uses the existing profile, organization, stack resolution, token refresh
and approval mechanisms. `--confirm` bypasses the write confirmation.

```bash
fctl stack demo --list
fctl stack demo --stack <stack-id> --type marketplace-payouts
fctl stack demo --stack <stack-id> --type crypto-custody --rate 100 --duration 10m
```

## Recipe catalog

The catalog contains all runnable schemas published in the documentation at
revision `2022dbe964864777d92ed6ed55f05671f2ac017a`. The source path and revision
are recorded in each embedded YAML. `recipes/*.yaml` are
compiled into the binary; running the command does not fetch documentation.

| `--type` | Official templates | Scenario |
| --- | ---: | --- |
| `bnpl-lending` | 10 | Financing, interest, fees, collection, returns, write-off and recovery. |
| `crypto-custody` | 16 | Fiat/crypto deposits, buy/sell trades, withdrawals, returns and compensation. |
| `issuing-financial-host` | 14 | Card authorizations, incremental/partial approvals, capture, refunds and disputes. |
| `marketplace-payouts` | 10 | Order capture, escrow, seller payouts, FX, refunds, chargebacks and reconciliation. |
| `neobank-fbo` | 18 | Deposits, transfers, cards, ACH withdrawals, advances and treasury movements. |
| `omnibus` | 5 | Identified/unidentified deposits, suspense resolution and reserved/settled payouts. |
| `payment-acceptance` | 4 | Authorization, acquirer settlement, refund and chargeback. |
| `rideshare-tutorial` | 4 | Ride booking, payment confirmation, fee split and driver payout. |
| `stablecoin-issuer` | 13 | Mint, transfer, redemption, returns, reserve rebalancing and yield. |
| `stablecoin-operations` | 7 | Payment/onramp, mint instruction/confirmation, burn and fiat off-ramp. |

These ten recipes contain 101 official transaction templates. The docs' three
chart-only illustrations (`ecommerce-platform`, `lending-platform` and
`multi-currency-wallet`) have no transaction scripts and are not offered as
runnable demos. Studio-only recipes absent from the documentation are also outside
this catalog.

## Command contract

| Flag | Default | Behavior |
| --- | --- | --- |
| `--list` | `false` | Lists recipes without authentication or network access. |
| `--type` | none | Required when running; accepts the catalog IDs above. |
| `--rate` | `1` | Global submission ceiling per second; finite, positive, at most 10000. |
| `--concurrency` | `0` | Maximum in-flight requests; 0 selects `min(512, max(1, ceil(rate / 10)))`. Values 1 through 1024 override it. |
| `--duration` | `0` | Duration after setup; zero runs until interrupted. |
| `--stack` | profile resolution | Uses the standard stack targeting rules. |
| `--confirm` | `false` | Skips the write confirmation. |

Each invocation creates a distinct `demo-<type>-<runID>` ledger and never adopts
an existing ledger. Before any write, `/api/ledger/_info` selects Ledger v2 or v3.
Both the v2 `data` envelope and the flat v3 response are supported. Missing or
unsupported versions fail before mutation.

Setup differs by API:

- Ledger v2 installs schema version `v1`, including its queries and transaction
  templates. The demo requests `experimental-interpreter` for every v2 template.
- Ledger v3 creates strict account types from the chart, including implicit empty
  leaves and explicit `.self` accounts. It saves official scripts at Numscript
  library version `1.0.0` and executes them through `scriptReference`. The adapter
  preserves references, variables, metadata and idempotency keys. It applies chart
  metadata defaults to concrete accounts involved in each script. The v2 query
  templates are not installed on v3.

The docs URL's deployment/license options do not establish compatibility with a
particular deployed server. V2 is covered by local interpreter and HTTP contract
tests; those checks do not replace testing against a deployed v2 server.

## Scenarios and opening funds

Omnibus randomly interleaves deposits, suspense resolution and payouts across
twenty clients per stream, one bank and one platform, using `EUR/2`. It caps
withdrawals by acknowledged available funds and each pending queue at 100 items.
It preserves the recipe's intentional bank overdraft and does not use `world`.

The other recipes repeat complete ordered lifecycle plans. Amounts vary between
cycles. Actor IDs are stable within a stream; order, authorization, conversion,
plan and business-event IDs change each cycle. Return/compensation metadata points
to actual earlier transaction references. Fees and crypto amounts use the asset's
integer scale, including `USD/2` and `USDC/6`. Variables derived by Numscript are
not supplied as external inputs.

Two recipes need explicit simulated opening funds because their documented
scripts do not provide a funding operation:

- `issuing-financial-host` funds the cardholder before exercising authorization
  branches. Its cycle returns the cardholder's spendable balance to zero.
- `neobank-fbo` funds the corporate operating account once per stream, enabling
  the documented FBO/corporate treasury movement pair.

These inline transactions are marked `demo.template=DEMO_BOOTSTRAP`; they are not
presented as official templates. The effective chart declares their `world`
source. The original embedded YAML remains unchanged. Rideshare also uses `world`
as its documented external source/sink.

All writes are synthetic Ledger entries. No bank, blockchain or Payments
connector is called. A stopped demo can leave holds, outstanding installments,
suspense balances or pending payouts; this is expected lifecycle state.

## Concurrency, progress and cancellation

Each stream has at most one request in flight and advances only after the server
acknowledges it. With concurrency greater than one, actor segments and transaction
references receive a `worker_NNN` namespace. Fixed platform accounts remain the
recipe's shared aggregation accounts; each lifecycle spends only its own prior
funding. Tests also interleave independent streams against one shared balance store.

A global token bucket enforces `--rate` across all streams. It retains at most
one millisecond of tokens, with a minimum of one transaction; it does not replay
an unbounded backlog after a stall. Concurrency hides network latency, but the
configured rate remains a ceiling, not a throughput guarantee.

Progress goes to stderr once per second: ledger, acknowledged transactions,
errors, interval rate, target rate, concurrency, in-flight requests and last
executed template. The final stdout summary reports the average acknowledged
rate and interrupted request count; `--output json` uses the standard `data`
envelope. List JSON contains recipe names and descriptions without run counters.
A failed stream renders its final summary after draining, then returns its original
error; the command still exits unsuccessfully. Authentication and debug
output retain the CLI's existing behavior.

Ctrl+C, SIGTERM and duration expiry cancel requests. Each request has a 30-second
timeout. A write error stops scheduling, cancels outstanding requests and drains
their results before returning. Acknowledged writes still count, including late
successful responses. Interrupted writes are counted in the summary, with one
aggregate notice that their commit status is unknown.

Each transaction carries a reference and `Idempotency-Key` derived from the run,
stream and sequence, plus `demo.run`, `demo.type` and `demo.template` metadata.
No automatic retry is configured; the first failed write stops the run. The keys
identify each write even if its response is lost. There is no persisted checkpoint
or resume mode: a second invocation creates another ledger. If a
response was lost, inspect the retained ledger by reference before manual action.
Setup failures also retain resources already created. No automatic deletion occurs.

## Maintainer verification

Ordinary tests remain offline. The Numscript evaluator runs every official
script through repeated cycles, with both isolated and shared stream balances.
Tests check exact variables, positive postings, funds conservation, chart/regex
matching, references, metadata defaults, pacing and cancellation/draining. HTTP
contract tests exercise the generated SDK for both API versions.

Live tests are opt-in and create retained demo ledgers. Run them only against an
intended test stack, with its normal authenticated profile:

```bash
FCTL_DEMO_LIVE_ORGANIZATION=<organization> \
FCTL_DEMO_LIVE_STACK=<test-stack> \
FCTL_DEMO_LIVE_MAJOR=v3 \
go test ./cmd/stack/demo -run '^TestCatalogLive$' -count=1 -v -timeout=5m
```

Use `FCTL_DEMO_LIVE_MAJOR=v2` for a v2 target. An unexpected API is rejected before
writes. `FCTL_DEMO_LIVE_PROFILE` optionally selects a profile. Live validation
executes every official template, repeats each planned cycle and verifies the
committed transaction count and template coverage through a fresh read.
