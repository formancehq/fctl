# Local Stack SDK

FCTL uses the complete generated Stack SDK in `pkg/client`. The root module
replaces `github.com/formancehq/fctl/v3/pkg/client` with that directory. Building
FCTL no longer downloads `formance-sdk-go/v4` or the private Connectivity SDK.
Membership and Deploy Server retain their existing separate local SDKs.

## Generation

The Go generator configuration comes from [formance-sdk-go v4.0.0](https://github.com/formancehq/formance-sdk-go/tree/v4.0.0), the version previously used by FCTL. Generation uses Speakeasy **1.757.1** and Go generator **2.866.2**, matching that release's lock. The only `gen.yaml` changes set the package/module path to `github.com/formancehq/fctl/v3/pkg/client` and version to `0.0.0` for this local module.

The workflow selects `stacks-source@v4.0.0-beta.1` and generates every service. An OpenAPI overlay removes only `x-speakeasy-name-override` from `ledger_V2BaseBulkElement`: with the v4 naming configuration, that annotation is inherited by four bulk variants and produces duplicate Go fields. The overlay preserves their schemas and wire format. Registry and code-sample publication are omitted.

The [Stack release](https://github.com/formancehq/stack/releases/tag/v4.0.0-beta.1)
contains the corresponding public `generate.json` OpenAPI asset; its SHA-256 is
`5a4570a8cbb926d532b1f5ec332c17e67833a5340248171b5345408f8dc4e039`.
The workflow and generation locks record the generator, options and document
checksum. Review source/tag and generated changes together when upgrading.

From the repository root:

```sh
nix develop --command just generate-stack-client
nix develop --command just tests
nix develop --command just pre-commit
```

Generation requires existing Speakeasy authentication, as for the other local
SDKs. Ordinary builds and tests use the checked-in code and need no Speakeasy
credentials or access to the private Connectivity repository. The dedicated
`stack-speakeasy` Nix package keeps Membership/Deploy Server on their own pinned
generator. Stack generation uses Go 1.25.10 for the official generator's bundled
Staticcheck; its importer does not support the repository's Go 1.27 export format.
FCTL builds and tests still use the repository toolchain.

The generated `pkg/client/docs` reference and generated SDK README, usage and contribution guides are excluded from Git; FCTL maintains its own documentation. Do not edit generated Go files. Change the source pin or generator configuration
and regenerate. Inspect the generated diff and locks when changing a source or
configuration, then repeat generation and check for unexpected drift. The source
is selected by its release tag: `--frozen-workflow-lockfile` cannot be used with
`--skip-upload-spec`, which intentionally leaves no uploaded source reference.
`just tests` covers both the CLI and the nested SDK module; the existing CI Test
job invokes that recipe. `just pre-commit` includes SDK regeneration.

## Consumer contracts

The Stack SDK owns routes, operation models and transport serialization.
The other services retain their command hierarchy. Their callers use the current
SDK property names (`data`, `cursor`, `metadata`, and typed union members) while
retaining FCTL's output envelopes. Ledger `server-infos` now calls the SDK's
current discovery route `/api/ledger/_/info`, returning the same CLI `server` and
`version` fields. Servers exposing only the former `/api/ledger/_info` route need
an upgrade before that command can be used; no legacy route fallback is provided.
Ledger schema commands preserve query parameters accepted by Ledger even when
the generated union expects an additional nested discriminator.
Payments raw provider objects now retain their fields in JSON output; the old SDK
decoded these objects as empty structs.
The generated Ledger v3 surface does not by itself add Ledger v3 CLI commands.

Local transport tests do not establish a supported live-server version matrix.
