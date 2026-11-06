# Local Stack SDK

FCTL uses the complete generated Stack SDK in `pkg/client`. The root module
replaces `github.com/formancehq/fctl/v3/pkg/client` with that directory.
Membership and Deploy Server retain their separate local SDKs.

## Source and generation

The workflow reads the committed `openapi/stack.json` snapshot. It was composed
from Stack `072263e3` (the local #2018 candidate), with these inputs resolved at generation:

| Component | Source revision |
| --- | --- |
| Ledger v2 (`release/v2.4`) | `f77eb05b129038c21071806c0f90af03bdc0e369` |
| Ledger v3 (`main`) | `8e30edf29e2586aefaeed7e0bc871936e5978bb7` |
| Connectivity (#526) | `9cbe55bdd0b00817f0bfaba3f8e07bdab256ffd0` |

Other component versions come from that Stack revision's `Justfile`.
The snapshot SHA-256 is
`7f4f4b0cab4c67380ffd56b5c8dc23db0a984890185846012d3e90b3358b1df1`.
It includes Connectivity's `errorCode`/`errorMessage`, string `details`, and
raw connection `status.lastError`. Ledger v2 includes the corrected query-template
contract from #2501: templates discriminate on their own `resource`, with flat
common parameters and separate volume parameters.

Generation uses Speakeasy **1.757.1** and Go generator **2.866.2**, matching
`formance-sdk-go v4.0.0`. Its Go options are preserved, including
`nameOverrideFeb2026: false`; the local module identity and version differ.
The bulk naming overlay removes `x-speakeasy-name-override` from
`ledger_V2BaseBulkElement` to avoid duplicate generated union fields. It changes
naming only. Registry and code-sample publication outputs are omitted.

The Connectivity error overlay omits error-body schemas from SDK generation only,
including the health probe’s explicit 500 response. The canonical OpenAPI snapshot
still documents their JSON bodies. Combined with per-operation `4XX`/`5XX` error
classification, this generates `sdkerrors.SDKError` for Connectivity HTTP failures,
preserving status, body bytes and the raw response. Commands return these errors
directly; no response-error converter is needed. Other components retain their
existing typed errors and generator settings.

```sh
nix develop --command just generate-stack-client
nix develop --command just tests
nix develop --command just pre-commit
```

Ordinary builds/tests use checked-in code. Generation requires existing
Speakeasy authentication. The `stack-speakeasy` Nix package isolates the Stack
pin from Membership/Deploy Server. Stack generation uses Go 1.25.10 for the
bundled Staticcheck; FCTL uses its repository toolchain.

To refresh the source, generate `releases/build/generate.json` in an isolated
checkout of the intended Stack revision, review its resolved component inputs,
and replace `openapi/stack.json`. Update the revisions and checksum above,
regenerate, inspect the SDK/lock diff, and repeat generation to check drift.
Ledger-specific validation in the selected Stack revision assumes the earlier
query-params union. The composed document was obtained with that obsolete
validation relaxed; validation-only changes do not alter the generated input.
SDK generation/tests do not qualify Stack for release.

Do not edit generated Go files. Generated SDK reference docs and usage guides
are ignored; FCTL maintains this document. `just tests` covers both the CLI and
nested SDK. `just pre-commit` includes regeneration. `--skip-upload-spec` keeps
generation local and cannot be combined with a frozen workflow lock.

## Consumer contracts and limits

Ledger `server-infos` retains the original `/api/ledger/_info` route and `data`
envelope through `Ledger.V1.GetInfo`. Import resume reads the latest log ID
through the SDK without a response-body fallback. Contract tests cover schema
logs, IDs above 2^53, the four plain query-param resources, and both info models.

Ledger schema commands use the generated SDK directly; no raw-params adapter
or response-body fallback is needed. The typed query-template union preserves
array expansion, time bounds, and volume `groupBy`/`insertionDate`. Regression
tests cover all four resources through get/list/insert, schema logs, YAML input
normalization, parent-resource validation, optional params, and run overrides.
A redundant nested `params.resource` is tolerated without losing time bounds.
The original file/URL loading logic is unchanged; local tests do not exercise
authenticated command execution against a deployed API.

Payments raw provider objects retain their fields in JSON output, but generic
JSON numbers use `float64` and are not lossless above 2^53. Compatibility with
historical Payments connector configs and Wallets servers is not established by
local model tests. The generated Ledger v3 surface adds no v3 CLI commands.

Local transport tests do not establish a live-server compatibility matrix.
