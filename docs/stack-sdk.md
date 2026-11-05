# Local Stack SDK

FCTL uses the complete generated Stack SDK in `pkg/client`. The root module
replaces `github.com/formancehq/fctl/v3/pkg/client` with that directory.
Membership and Deploy Server retain their separate local SDKs.

## Source and generation

The workflow reads the committed `openapi/stack.json` snapshot. It was composed
from Stack `2615a186` after #2017, with these moving inputs resolved at generation:

| Component | Source revision |
| --- | --- |
| Ledger v2 (`release/v2.4`) | `275962012684950622efbe0875eb02dbdd323b5c` |
| Ledger v3 (`main`) | `c523c431b3ce6fd4070e8d7293e961be8dfa6749` |
| Connectivity (`main`) | `6c6f971e3f638a009a9ee4446d1daad3a9ec1c8f` |

Other component versions come from that Stack revision's `Justfile`.
The snapshot SHA-256 is
`e18983308bfe4d5387090c67522d414fba76c81da91103673c3a64f94fcc6a46`.
It includes Connectivity's `errorCode`/`errorMessage`, string `details`, and
raw connection `status.lastError`. Ledger #2501 is not included.

Generation uses Speakeasy **1.757.1** and Go generator **2.866.2**, matching
`formance-sdk-go v4.0.0`. Its Go options are preserved, including
`nameOverrideFeb2026: false`; the local module identity and version differ.
The bulk naming overlay removes `x-speakeasy-name-override` from
`ledger_V2BaseBulkElement` to avoid duplicate generated union fields. It changes
naming only. Registry and code-sample publication outputs are omitted.

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
The current Stack validator still assumes every V2 query-params union member
has a resource constant and rejects the valid plain-params member. This upstream
gate failure must be assessed separately; SDK generation/tests do not qualify
Stack for release.

Do not edit generated Go files. Generated SDK reference docs and usage guides
are ignored; FCTL maintains this document. `just tests` covers both the CLI and
nested SDK. `just pre-commit` includes regeneration. `--skip-upload-spec` keeps
generation local and cannot be combined with a frozen workflow lock.

## Consumer contracts and limits

Ledger `server-infos` retains the original `/api/ledger/_info` route and `data`
envelope through `Ledger.V1.GetInfo`. Import resume reads the latest log ID
through the SDK without a response-body fallback. Contract tests cover schema
logs, IDs above 2^53, the four plain query-param resources, and both info models.

Ledger schema commands retain a raw-params adapter. The selected v2 contract
accepts plain params, but legacy union members still silently discard time
bounds when a redundant nested `params.resource` is present. The adapter
preserves those existing inputs and returned values; its regression test covers
both insert and get. It remains necessary with the selected source.

Payments raw provider objects retain their fields in JSON output, but generic
JSON numbers use `float64` and are not lossless above 2^53. Compatibility with
historical Payments connector configs and Wallets servers is not established by
local model tests. The generated Ledger v3 surface adds no v3 CLI commands.

Local transport tests do not establish a live-server compatibility matrix.
