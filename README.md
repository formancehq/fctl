# Formance Control CLI (fctl v4)

`fctl` is the Formance command-line interface (repository profile: CLI).
The v4 implementation embeds Cloud, Auth and Ledger plugins with a shared
connection boundary for local services, OAuth2 client credentials and Cloud login.
The Go module is `github.com/formancehq/fctl/v4`.

## Current commands

```bash
go run . --help
go run . version
go run . --version
go run . completion bash
go run . --auth-mode none --ledger-url http://localhost:9000 ledger list
go run . connections --help
go run . login
go run . cloud organizations list
go run . cloud stack list --organization ORG_ID
go run . --organization ORG_ID --stack STACK_ID ledger list
```

Development builds report `v4.0.0-dev`. Release builds receive their version,
commit and build date through GoReleaser. No v4 release is published by this
source change.

The Auth module uses its public Go client. Ledger targets `release/v3.0`,
with HTTP business routes under `/v3`. See [module commands](docs/modules.md)
and [connection setup](docs/connections.md). Cloud management lives under
`cloud`, with stack administration under `cloud stack`. See the
[Cloud command boundaries](docs/cloud.md) for restored proxy, MCP and token
helpers, and experimental Deploy Apps. Connectivity is deferred.

`ledger list` returns all ledgers. Account, transaction and log lists continue
with `--after` using the last address or ID; only Ledger index inspection uses
opaque `--cursor` tokens. See [Ledger pagination](docs/modules.md#ledger).

The v3 command tree and profile store are not migrated automatically. Users of
v3 scripts must adapt commands and request schemas or retain a v3 binary.
The v4 profile store is separate and direct service access needs no Cloud login.

`fctl login` defaults to the public Cloud, opens the browser and saves a Cloud
connection after successful authentication. Use `--no-browser` for a headless
session or `--issuer` for another environment. Login does not require a stack;
service commands resolve their target from flags, saved defaults, unique access or
interactive selection.

Output defaults to readable tables on a terminal and JSON in pipes or files.
Use `-o json` for all fields and automation; table lists summarize scalar columns
without repeated output reminders. `--color auto|always|never` controls styles.
See [output and diagnostics](docs/output.md), including `NO_COLOR` and `-d`.

Commands offer forms and searchable selections when required input is missing.
Supply arguments and flags for direct execution, or use `--no-input` for scripts.
See [interactive command input](docs/interaction.md).

Plugins expose manifests and execution through a public SDK. The core adapts
their manifests to Cobra; service plugins do not depend on Cobra or core internals.
See the [plugin contract](docs/plugins.md). This
CLI uses explicit composition without Fx. Proxy and MCP commands run until
cancellation; they use the host's connection and authentication boundary.

## Development

Use the Go version and toolchain declared in `go.mod`, or the repository's Nix
development shell:

```bash
nix develop --impure
just pre-commit
just tests
go build ./...
```

`just tests` runs with the race detector. `just generate` runs Go generators;
the removed Membership and deployment SDKs are no longer generated.
`just completions` updates the shell scripts shipped in release archives.

The repository includes the `formance-fctl-plugin` agent skill for developing
service plugins through the public SDK. Its sources are versioned under
`.agents/skills/`, where Codex discovers them for this repository, and included
in release archives. See [agent skill usage and installation](docs/agent-skills.md).

## Documentation

- [Technical documentation](docs/README.md)
- [Formance documentation](https://docs.formance.com)

## License

MIT License. See [LICENSE](LICENSE).
