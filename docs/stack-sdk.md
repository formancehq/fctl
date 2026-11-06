# Local Stack SDK

FCTL uses the complete generated Stack SDK in `pkg/client`. The root module
replaces `github.com/formancehq/fctl/v3/pkg/client` with that directory.
Membership and Deploy Server retain their separate local SDKs.

## Source and generation

The workflow reads the exact versioned
[Stack v4.0.0-beta.3 release asset](https://github.com/formancehq/stack/releases/download/v4.0.0-beta.3/generate.json)
directly. No composed OpenAPI snapshot is vendored in FCTL.
The release tag points to Stack commit
`a476c6c75c480b91626c64919f7450a3625a16ab`. The asset filename is `generate.json`
(its release label is `openapi.json`). Its SHA-256 is
`4395810e7958f1bc5a63337dca5344441d361240ed3b99d9b994f6571509f48f`.
This published artifact is the authoritative generation input; moving component
branches are not fetched when regenerating FCTL. The asset is downloaded only
during generation; ordinary builds and tests use the committed generated SDK.

The asset includes Connectivity's `errorCode`/`errorMessage`, string `details`,
and raw connection `status.lastError`. Ledger v2 query templates discriminate
on their own `resource`, with flat common parameters and separate volume
parameters.

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

Ordinary builds/tests use checked-in code. Generation requires network access
to the versioned release asset and existing Speakeasy authentication. The `stack-speakeasy` Nix package isolates the Stack
pin from Membership/Deploy Server. Stack generation uses Go 1.25.10 for the
bundled Staticcheck; FCTL uses its repository toolchain.

To refresh the source, select a published Stack release and update the exact
`generate.json` asset URL in `pkg/client/.speakeasy/workflow.yaml`. Verify the
asset digest and update the tag, commit and checksum above. Regenerate, inspect
the SDK/lock diff, and repeat generation to check drift. SDK generation/tests do
not qualify Stack for release or establish a deployed compatibility matrix.

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

Connectivity HTTP `4XX`/`5XX` responses are returned through the SDK error
channel. Consumers must handle the returned error before using the response;
the previous `response.Error` field is no longer generated. JSON application
errors carry `errorCode`, `errorMessage` and string `details`. Non-JSON and
bodyless responses use `sdkerrors.SDKError`, retaining the HTTP status and body.

Payments raw provider objects retain their fields in JSON output, but generic
JSON numbers use `float64` and are not lossless above 2^53. Compatibility with
historical Payments connector configs and Wallets servers is not established by
local model tests. The generated Ledger v3 surface adds no v3 CLI commands.

Local transport tests do not establish a live-server compatibility matrix.
