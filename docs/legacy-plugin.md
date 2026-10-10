# Historical service plugin

The legacy plugin preserves service commands from fctl v3 in the independent
`github.com/formancehq/fctl/misc/fctl-plugin` module. Its source baseline is
`e00243b3e2e56aae6a09d7010b0c17890134388c` on the historical `main` branch.
The command implementations are adapted to the public SDK; they do not import
Cobra, core packages, profiles, token storage or terminal libraries.

## Provider selection

The first implementation embeds the SDK service facades in fctl and also builds
an independent `fctl-plugin-legacy` executable. Both call the same implementations.
The executable exposes a `legacy` command tree with per-command service boundaries.
A future distribution change can move these facades into the executable without
porting the service operations again.

The core resolves the authenticated endpoint and inspects `/_info`. For Cloud
stacks it also reads the exact stack compatibility line from Membership.
Stacks in lines v1 through v3 can select legacy service facades; v4 prereleases
and later select modern plugins. Auth versions can be identical across stack
lines, so Auth version alone cannot identify a historical Cloud stack.

For direct service connections, selection uses the service API family. An
installed modern plugin or `FCTL_PLUGIN_CATALOGUE` selects the modern provider.
A prepared modern provider stays modern after a service upgrade; its version
guard or catalogue resolution rejects an unavailable release. A missing modern release, failed download or invalid
checksum never activates a legacy fallback.

| Legacy service | Supported API majors |
| --- | --- |
| Ledger | 1, 2 |
| Auth | 1, 2 |
| Payments | 1, 2, 3 |
| Orchestration, Reconciliation, Wallets, Webhooks | 1, 2 |

Compatibility belongs to the plugin. Its own revision (`1.0.0`) is independent
of the target service version. An unknown development version is rejected.
The service version is checked again before each legacy execution, including
choice queries. The Cloud resolver also checks the stack line before execution.
This prevents a cached historical command from mutating an upgraded v4 stack.
Historical and current `/_info` envelopes are read by `pluginsdk.ServiceVersion`;
conflicting or absent version values are errors.

## Metadata cache

The core records selected providers under the private configuration directory:
`plugins/selections/<sha256-target-and-service>.json`. Each record includes the
profile, organization, stack, endpoint, provider, service version, stack line
and plugin revision. Credentials and transient gateway URLs are excluded.
Records are atomically replaced, bounded and validated. Another target never
inherits a record. The modern OCI locks and binary cache keep their existing
format and are not replaced by this metadata.

`fctl plugins selections` shows these records without contacting the service.

Help and completion use local metadata. `fctl legacy --help` always describes
the historical bundle without contacting a service. Normal module roots such
as `ledger` select their prepared facade for the exact target. Execute a
command once to prepare a previously unseen target.

```sh
fctl ledger list --organization ORGANIZATION_ID --stack LEGACY_STACK_ID
fctl legacy auth clients list --organization ORGANIZATION_ID --stack LEGACY_STACK_ID
fctl ledger list --organization ORGANIZATION_ID --stack V4_STACK_ID
```

## Build and validation

```sh
nix develop --impure --command just build-legacy-plugin
nix develop --impure --command just pre-commit
nix develop --impure --command just tests
```

The root Just recipes include the separate SDK and legacy modules. GoReleaser
builds the independent executable for Linux, macOS and Windows on amd64 and
arm64, in its own archive. This change does not publish a release or advertise
compatibility ranges in the exact-version product catalogue.

Mutation payloads and Numscript stay in the plugin. The host reads `--data`
inline JSON, `@file` or stdin and renders the response. Historical file operations
use the SDK body/result boundary instead of accessing the user's filesystem in
the plugin. Delete/revert operations require explicit confirmation.

## Validation record

See [the sandbox validation record](legacy-validation.md) for the tested Stack
v3.2 module versions, operations, authentication modes and remaining limits.
