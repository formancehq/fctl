# V4 implementation validation

## Scope and evidence

This record summarizes the sanitized validation reports from October 8–9, 2026.
It records completed checks and remaining gaps separately. Historical failures
and preparation fields in the reports do not override their final status.

The approved live target was stack `bwxm` in `eu-sandbox`, targeting
`v4.0-beta`. Observed services were Auth `v2.5.1` and Ledger
`3.0.0-beta.5` (runtime commit `36d580949`). These component versions do not
establish an exact final v4 runtime release.

The current Ledger source contract is `release/v3.0` at
`0f4656d1efbcac42705839daccb34012e70d783a`. The deployed beta has a different wire
contract; source compatibility and deployed behavior must be assessed separately.

| Evidence | Final result |
|---|---|
| `/tmp/fctl-ledger-live-report.json` | `VALIDATED_AND_CLEANED_UP`; 31 positive command paths; no current open findings |
| `/tmp/fctl-auth-live-report.json` | `COMPLETE_WITH_GAPS`; 11 of 12 positive command paths; cleanup complete |
| `/tmp/fctl-auth-modes-report.json` | Five cases `PASS`, including expected authentication failures |
| `/tmp/fctl-cloud-read-report.json` | 18 cases `PASS`, one `FAIL` |
| `/tmp/fctl-cloud-crud-20261008/report.md` | 46 commands, 44 `PASS`, 29 assertions passed; one entitlement refusal and one preservation assertion failure |
| `/tmp/fctl-stack-tools-report.json` | Five cases `PASS`; MCP initialization `BACKEND_UNAVAILABLE` |
| `/tmp/fctl-presentation-live-report.json` | Four real PTY cases `PASS`; no backend writes |
| `/tmp/fctl-cloud-lifecycle-report.json` | `VALIDATED_AND_RETAINED_READY`; temporary organization absent, stack retained `ACTIVE`/`READY` |
| `/tmp/fctl-apps-live-report.json` | `PASS_WITH_COVERAGE_LIMITS`; 35 successful calls, five expected errors, 15 assertions passed; cleanup verified |

These paths identify local evidence artifacts, not files shipped with the
repository. Later validation results and fixes are recorded below separately
from the historical report results.

## Ledger live validation

All 31 Ledger command paths were positively validated, including ledger and
index deletion. The campaign recorded 338 live command cases. Its final
follow-up has 105 passing assertions, zero failures and zero blocked assertions;
cleanup adds 48 passing assertions. All dedicated ledger and index fixtures were
deleted and their absence verified. The stack was not deleted or changed by
Ledger cleanup.

The final checks cover ledger CRUD, accounts, transactions, metadata, balances,
statistics, indexes, logs and bulk operations. Idempotent replay, conflicting
keys, bulk stop/continue behavior, reversals, filters and pagination were
exercised. Earlier pagination findings and the metadata fixture oracle error
were resolved by subsequent validation; the historical failure counters remain
in the report for traceability.

The deployed beta accepts posting amounts as JSON numbers and rejects quoted
decimal strings. The exact posting value `184467440737095516170` was verified.
Metadata retained the exact integer `9007199254740993`, the unsigned 64-bit
maximum and the signed 64-bit minimum. The CLI preserves raw JSON tokens rather
than converting these amounts through floating point.

Deployed metadata conversion omits null values; the initial null-preserving
oracle was corrected. Ledger log continuation uses `payload.apply.log.id`, not
the outer global log sequence. The full paginated log oracle passed after this
correction. The deployed asset precision range is 1 through 255; a valid mixed
precision fixture passed after the rejected `EUR/0` case and its changed-body
idempotency conflict.

Positive `scriptReference` use remains unverified because the campaign did not
provision a library. High transaction IDs and actual server 5xx responses were
not safely generated. Passing command-path coverage does not remove these
variant limitations.

## Auth and authentication modes

Auth completed 63 recorded cases with no live functional failures. Eleven
positive command paths passed: service information, discovery, client CRUD,
secret list/create/delete and users list. Positive `auth users show` remains
blocked: the service returned no users and no synthetic user or invitation was
created. The missing-user 404 case passed. The runner's final exit code of 1
reflects `COMPLETE_WITH_GAPS`, even though cleanup succeeded.

Client updates verified omitted-value preservation and explicit option clearing.
Confirmation checks and readbacks verified secret revocation and client deletion.
Both created secrets were revoked, the fixture client was absent, and the private
credentials file was removed. A harness assertion that initially rejected the
server's stored secret hash was corrected without replaying completed mutations.

Client and secret readbacks exposed a nonempty stored `hash` field but no clear
secret. The CLI preserves the raw server JSON. The report classifies this as an
Auth read-model observation, not a missing CLI command. Local source explains
the serialization mechanism, but the deployed Auth commit was not verified.
Secret values are not included in the sanitized report.

All five authentication-mode cases passed: anonymous discovery, rejection of
anonymous protected Ledger access, client-credentials Auth and Ledger reads,
and rejection of an invalid client secret. Diagnostics hid credentials in all
five cases. The initial approved browser login completed, and cached root grants
were exercised earlier. The latest service reports used the existing profile
and cached stack grant; they do not establish a new device authorization or
fresh browser login.

## Cloud reads and CRUD

Cloud reads passed for identity, invitations list, organization list/describe/
history, users, policies, OAuth clients, applications list, regions and versions,
and stack list/info/version/history/users/modules. The read report records one
unsuccessful `organizations authentication-provider show` case as `FAIL`. Initial
validation confirmed an expected missing-provider 404, but the raw read report
contains no error reason. The separate isolated CRUD campaign positively
validated provider configure/show/delete and the expected missing-provider 404
after deletion.

In the isolated QA organization, policy CRUD and scope add/remove passed. OAuth
client CRUD passed, with final absence checked from captured cursor data. The
dummy Google authentication provider was removed. Successfully created nested
fixtures were deleted. The lifecycle campaign subsequently deleted the temporary
QA organization and verified its absence.

Private region creation was refused by the paid-feature entitlement check
(HTTP 400 `VALIDATION`). No private region was created and the region IDs were
unchanged. This is a blocked positive validation, not a successful region CRUD
campaign. Users link/unlink were skipped because no own or synthetic user was
available; no invitations were sent.

The earlier organization name update and restoration passed, but strict preservation of
`defaultPolicyID: null` failed: Membership updates default it to Guest policy
ID `4`. The report's source review confirmed explicit API defaulting when the CLI
sent preserved null in its request. This is not established as semantically neutral:
the source gives future users different scopes under Guest than under an unset
policy. Existing member permissions are not recomputed by this update. The
future-user effect was not tested live. The original organization name and
domain were preserved or restored; the default policy was not restored to null
in that campaign.

The CLI protection was subsequently fixed: an organization update with an unset
or zero default policy is refused before PUT unless an explicit positive policy
ID is selected. Regression tests cover this refusal and preservation of an
existing positive policy. The earlier live preservation failure is a historical
finding addressed by this guard; no subsequent live preservation result is
recorded here.

Applications list passed with the latest rebuilt CLI, including an empty cursor.
The scope broker backend audience handling was fixed and passed earlier tests.
Successful private region operations and user link/unlink remain outside the
completed evidence.

## Applications

The Apps campaign completed with `PASS_WITH_COVERAGE_LIMITS`: 35 successful
calls, five expected errors and 15 passing assertions, with no failed assertions
or code changes. Application creation, variables, manifests, versions,
pagination, manifest binding/unbinding and downloads were verified. Variable
redaction, distinct second pages and exact downloaded bytes passed. Cleanup
verified that the owned application, manifest and variables were absent; no
owned resources remained.

No deployment was created: actual infrastructure creation was outside the
approved QA scope. Deployment lists were empty before and after the campaign,
and missing-deployment reads returned the expected HTTP 403 with exit code 1
and empty stdout. Positive deployment show/logs/download paths remain
unverified. Deployment creation and its polling with `--wait` were not tested.
Metadata cleanup does not establish a working deployment lifecycle.

## Stack lifecycle

The lifecycle campaign completed with `VALIDATED_AND_RETAINED_READY`. Stack
disable/enable, deletion without force and restoration passed. Deletion without
force retained resources with stack state `DELETED` and status `DISABLED`;
restoration returned the stack to `ACTIVE`/`READY`. The October 8 campaign did
not test forced deletion. The later form campaign force-deleted only its own
new, empty sandbox fixture.
The temporary organization was deleted and absent from subsequent lists. Ledger
fixture absence was verified again after restoration.

Same-version upgrade passed as an idempotent CLI no-op on a ready stack; no
backend version transition was tested. Two independent reviews found the same
P2 issue: this path bypassed requested waiting when the stack was `PROGRESSING`.
The issue is resolved by `waitExistingStackVersion`: default waiting requires
both `READY` and the requested version, `--no-wait` returns the current stack,
disabled stacks are rejected, and cancellation is honored. Meaningful regression
tests cover these cases; the race-enabled `TestStack` checks passed. No new live
version transition is claimed.

Restoration briefly returned `READY` before the stack became `PROGRESSING` again
and a Ledger list request returned HTTP 503 `NO_LEADER`. Subsequent read-only
checks confirmed recovery, with two successful empty Ledger lists and repeated
stack readiness. No mutation was retried. The final recorded stack is `ACTIVE`,
`READY`, reachable and synchronised at `v4.0-beta` in `eu-sandbox`.

## Stack tools

Personal token acquisition passed without reporting the credential. The proxy
returned Ledger information for `3.0.0-beta.5` and replaced incoming credentials.
Host and origin rejection each returned HTTP 403. Proxy shutdown was classified
`PASS` by the report with exit code 1.

Before disablement, the MCP module's backend status was `PROGRESSING`, with an explicit
`DependencyVersionMismatch`: effective Ledger version `v3.0.0-beta.5` must be
before `v3.0.0-0` to satisfy its dependency requirement. MCP initialization
could not complete on this backend. The recorded case is `BACKEND_UNAVAILABLE`: the backend returned
HTTP 404, surfaced as RPC error `-32000` while preserving the request ID.
The process exited 0, which does not establish a working MCP backend. No
successful initialization or MCP session is claimed.
The lifecycle campaign then disabled MCP and verified state `DISABLED` and
status `DELETED`; the retained stack recovered to `READY`.

## CLI output and errors

All four real PTY cases passed at a terminal width of 100 columns: help used
automatic ANSI color, `NO_COLOR` suppressed it, automatic region listing rendered
a colored ASCII table with wrapped fields, and explicit JSON remained valid and
free of ANSI even with color forced. These checks made no backend writes.
Explicit JSON output and non-TTY output tests also passed, including exact
preservation of large integers.

The actual `auth info` error reproduction produced three lines and exit code 1,
without listing target IDs. The full suite includes the regression check that
bounds target error output.

## External plugin SDK consumer

A separate consumer at `/tmp/fctl-plugin-consumer-5b2bsq26` compiled with
`GOWORK=off` and executed the Cloud, Auth and Ledger manifests. It preserved the
exact integer `9007199254740993`. Its `go list -deps` output contains no Cobra,
pflag or core `internal`/`cmd` dependencies. The repository's dependency tests
also passed.

## Repository race checks and coverage

The full race-enabled repository suite associated with
`/tmp/fctl-cloud-final4-coverage-20261008.out` passed with exit code 0, including
the bounded target error checks and same-version upgrade readiness regressions.
The upgrade fix was independently reviewed and the finding was closed.
Aggregate statement coverage computed from that artifact is **87.7%**.

The final `just pc` check passed with zero lint issues. The final GoReleaser snapshot
successfully built all six targets: Linux, macOS (`darwin`) and Windows, each on
amd64 and arm64. No release was published.

To reproduce the suite and compute coverage:

```bash
nix develop --impure --command go test -count=1 -race -timeout 120s \
  -coverpkg=./... -coverprofile=/tmp/fctl-v4-coverage.out ./...
nix develop --impure --command go tool cover -func=/tmp/fctl-v4-coverage.out
```

Snapshot builds do not establish signing, notarization, release publication or
deployment validation.
Snapshot version metadata still derives from the existing v3 Git tags; an
official v4 release requires a v4 tag. Development builds report `v4.0.0-dev`.


## Interactive command validation (2026-10-09)

Terminal forms were exercised in a 100-column, 24-line pseudo-terminal, with
stdin and stderr on the terminal and stdout captured separately. The checks
covered real keyboard navigation, searchable resource menus, cancellation,
confirmation defaulting to No, and JSON free of terminal controls with forced
color profiles. A local HTTP capture verified that the form sends exact JSON
numbers, including `9007199254740993`.

Live Cloud validation used organization `jdxmvkvwlyiy` and the retained v4 beta
stack `bwxm`. It covered:

- Ledger creation through all form steps, resource selection for stats,
  cancellation before creation, declined deletion, and confirmed cleanup.
- Auth client creation with metadata, client selection, secret creation, and
  removal of the dedicated client and secret.
- Stack creation through name, region and catalog selection in `eu-sandbox`;
  exact `v4.0-beta` was recorded and the temporary stack was verified `DELETED`.
- `auth info` without organization or stack flags: both menus selected the
  authorized sandbox and the service read completed.
- Ledger metadata editing with exact integer `9007199254740993`, Numscript
  transaction creation, reference readback, and verified fixture deletion.

A further terminal check used stdin, stdout and stderr on the same terminal.
Selecting the user ledger for `stats` produced the automatic table output;
`NO_COLOR` suppressed colors, and neither omitted-field counts nor JSON
reminders appeared.

The final full suite passed with the race detector and `-coverpkg=./...`.
Aggregate statement coverage in `/tmp/fctl-interactive-final-coverage.out` is
**88.3%**. `just pc` and the final lint check reported zero issues. An independent
review found no remaining P1 or P2 defects. A separate SDK consumer compiled and
executed all three plugin manifests without a core or Cobra dependency.

The local GoReleaser snapshot succeeded with
`goreleaser release --snapshot --clean --skip=publish,docker`: all six binaries,
archives, Linux packages and Homebrew cask files were generated. Docker images
remain unverified because the local daemon was unavailable. Nothing was
published.

The user ledger `toot` and the retained stack `bwxm` were preserved. No real
identity provider, invitation or production resource was changed by form QA.

### Ledger beta creation metadata

The sandbox runs Ledger `3.0.0-beta.5` (`36d580949`). Its creation HTTP and
protobuf models omit `metadata`, so that server ignores metadata in a create
request. The current `release/v3.0` contract persists it atomically after Ledger
[PR #2184](https://github.com/formancehq/ledger/pull/2184), included in beta.10.
The CLI's flat creation body is correct for that contract. HTTP server tests
and a terminal capture verified the field shape and exact numeric value.

The form explains the beta.10 prerequisite. On the older sandbox, use
`ledger metadata set` after creation; creation does not claim that the old
server persisted this field. The other creation options, including enforcement
mode, initial schema and account type models, exist in both revisions.

## External Ledger plugin pilot (2026-10-09)

The Ledger implementation and its contract tests now live in the public Go
module `github.com/formancehq/ledger/fctl-plugin`, on branch
`feat/fctl-ledger-plugin` at `885e135ef98a232aa9e4c05a5cb2895ce11c4238`.
The branch starts from `release/v3.0`. fctl retains an embedded factory shim and
can also execute the separately built plugin over the isolated SDK protocol.

A standalone Go consumer downloaded both public modules without a local
replacement and ran the Ledger manifest successfully. Its dependencies contain
neither the fctl core nor Cobra. The host retains authentication, HTTP diagnostics,
profiles, forms and rendering; the plugin owns Ledger commands and payloads.

The following checks passed:

- The complete host and SDK suites with the race detector, plus lint checks.
- Ledger's `agent-check`, plugin race tests and lint checks.
- Six-platform plugin packaging, both Ledger GoReleaser configurations and the
  fctl snapshot, including archives and Linux packages.
- Exact-version OCI resolution, executable integrity, cached metadata and
  offline help through actual plugin processes.
- Publisher rejection of stale manifests, mixed service versions and wrong
  revisions before publication.
- Four independent review regressions: implicit Cloud target preservation,
  cancellation without panic, publisher validation and help without a binary.

Fresh live reads observed Ledger `3.0.0-beta.10` (`b7c2ec613`, protocol 20) and
Auth `v2.5.1` on organization `jdxmvkvwlyiy`, stack `bwxm`. This supersedes the
historical beta.5 runtime observations above. A real terminal exercised the
five-step Ledger creation form and the Ledger selector for statistics. Creation
metadata persisted. A transaction and balance readback retained the exact
integer `9007199254740993`; JSON output remained parseable.

The dedicated ledger `fctlplugin20261009` was deleted, and a subsequent list
confirmed its absence. The stack and installed plugin remain available for
manual testing. This campaign did not create Auth fixtures or modify other
stacks.

No production release tag, production OCI publication or official default
catalogue was created. Snapshot versions still derive from existing v3 tags.
The optional Jev review was unavailable because `JEV_API_KEY` was unset;
independent review and repository checks completed instead.


## Product-owned Ledger plugin layout (2026-10-09)

The embedded Ledger adapter now imports
`github.com/formancehq/ledger/misc/fctl-plugin` at
`v0.0.0-20261009143954-060fabc5af8d` from the Ledger branch
`feat/fctl-shared-commands`. This is a public module pin, without a local
Ledger replacement. The plugin remains independent of fctl core, terminal UI,
profiles, credentials, and Ledger server internals.

The same command manifest runs through HTTP in fctl and through the native
gRPC adapter in ledgerctl. A local single-node Ledger was used to check fctl
listing and `after` pagination over HTTP, alongside ledgerctl's 38-case matrix
covering all 31 shared operations. The isolated node advertised
`3.0.0-beta.10`; these checks do not establish a deployed service release.
