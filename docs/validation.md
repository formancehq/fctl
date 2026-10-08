# V4 implementation validation

## Ledger reference and local server

The requested Ledger v3 release branch is named `release/v3.0` in the repository.
The CLI contract and local server validation use commit
`71b0feb549a7cbc22bddaca2859ae5d91a24d378` from that branch.

A server compiled from this revision was started locally without authentication.
The CLI completed ledger creation/listing, account reads, transaction creation,
idempotent replay, metadata updates/deletion, transaction reversal, balances,
statistics and server information. The amount `9007199254740993` remained exact
through creation, replay and account reads. Replay returned the original
transaction rather than creating another one; reversal restored a zero balance.
The local server was stopped after validation.

## Authentication and CLI integration

Automated HTTP fixtures exercise anonymous access and OAuth2 client credentials.
A local Cloud fixture signs RS256 ID tokens and serves OIDC discovery, JWKS,
Membership device authorization and Auth token exchange. Tests invoke fresh CLI
instances against the same private connection directory and cover login, Auth
and Ledger requests, cached tokens, logout and secret redaction.

The Cloud login fixtures separate the root Membership identity from stack-scoped
authorization. They check target selection, independent caches for multiple
stacks, browser callbacks and headless instructions. Failed or canceled login
must preserve existing profile settings, tokens and the active connection; new
profiles are written only after successful authentication. Browser launcher tests
use injected processes and do not open a real browser.

Device polling omits `scope`, matching the former CLI: Membership binds the
scopes during device authorization and does not split a space-delimited scope
field at its token endpoint. The fixtures reject polling requests that repeat
this field. Scoped requests still carry their `resource`.

HTTP debug tests cover `-d`/`--debug`, stdout/stderr separation, credential
redaction and unchanged request/response streams. Known OAuth error codes remain
visible; arbitrary error descriptions are redacted.

Regression tests cover concurrent profile updates, stale login/refresh results
after logout or replacement, refresh-token rotation, cancellation, request
limits, URL escaping, exact JSON numbers and bulk partial failures. Mutating
requests are not retried automatically.

These fixtures validate the implemented contracts. No live Cloud organization
or deployed Auth service was used. External plugins and Connectivity are outside
this implementation.

A live public Cloud smoke check completed discovery and device initiation, then
continued polling through `authorization_pending` responses. It was canceled
without opening a browser or writing a profile. User-approved authentication
and service requests against a live Cloud stack were not verified.

## Reproducible repository checks

The initial implementation checks passed with zero lint issues, a passing
race-enabled suite and 86.2% aggregate statement coverage. The plugin refactoring
is validated independently with the same checks below.
Its final race-enabled suite passed with 88.5% aggregate statement coverage and
zero lint issues. The six GoReleaser snapshot targets built successfully.
The default Cloud login and HTTP debug changes passed a fresh full race-enabled
suite with 89.0% aggregate statement coverage, zero lint issues and successful
builds for the same six snapshot targets.

The plugin dependency test inspects the complete production import closure and
rejects Cobra, pflag and core `internal`/`cmd` dependencies. A separate Go consumer
with `GOWORK=off` compiled and executed Auth and Ledger manifests and server-info
operations through the public SDK and an injected HTTP fixture. Its binary
dependencies exclude Cobra and the core implementation.

```bash
nix develop --impure --command just pc
nix develop --impure --command go test -race -timeout 60s ./...
nix develop --impure --command go test -count=1 -race -timeout 60s \
  -coverpkg=./... -coverprofile=/tmp/fctl-v4-coverage.out ./...
nix develop --impure --command go tool cover -func=/tmp/fctl-v4-coverage.out
```

The GoReleaser snapshot built archives for Linux, macOS and Windows on amd64 and
arm64. Publication, signing, notarization, system packages and Docker publishing
were skipped. Snapshot versions still derive from the existing v3 Git tags;
publishing v4 requires an explicit v4 release tag. This validation did not publish
a release.

An independent code review was performed. Jev Review was unavailable because its
API key was not configured.
