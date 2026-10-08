# Formance Control CLI (fctl v4)

`fctl` is the Formance command-line interface (repository profile: CLI).
The v4 implementation embeds Auth and Ledger modules with a shared connection
boundary for local services, OAuth2 client credentials and Cloud user login.
The Go module is `github.com/formancehq/fctl/v4`.

## Current commands

```bash
go run . --help
go run . version
go run . --version
go run . completion bash
go run . --auth-mode none --ledger-url http://localhost:9000 ledger list
go run . connections --help
```

Development builds report `v4.0.0-dev`. Release builds receive their version,
commit and build date through GoReleaser. No v4 release is published by this
source change.

The Auth module uses its public Go client. Ledger targets `release/v3.0`,
with HTTP business routes under `/v3`. See [module commands](docs/modules.md)
and [connection setup](docs/connections.md). Connectivity is deferred.

The v3 command tree and profile store are not migrated automatically. Users of
v3 scripts must adapt commands and request schemas or retain a v3 binary.
The v4 profile store is separate and direct service access needs no Cloud login.

Plugins expose manifests and execution through a public SDK. The core adapts
their manifests to Cobra; Auth and Ledger do not depend on Cobra or core internals.
See the [plugin contract](docs/plugins.md). This
CLI has no long-running service lifecycle, so composition does not use Fx;
review this choice if persistent background services are introduced.

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

## Documentation

- [Technical documentation](docs/README.md)
- [Formance documentation](https://docs.formance.com)

## License

MIT License. See [LICENSE](LICENSE).
