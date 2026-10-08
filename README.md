# Formance Control CLI (fctl v4)

`fctl` is the Formance command-line interface. This branch starts the v4
implementation with a minimal CLI foundation (repository profile: CLI).
The Go module is `github.com/formancehq/fctl/v4`.

## Current commands

```bash
go run . --help
go run . version
go run . --version
go run . completion bash
```

Development builds report `v4.0.0-dev`. Release builds receive their version,
commit and build date through GoReleaser. No v4 release is published by this
source change.

The previous service, Cloud, profile and interactive commands have been removed.
Ledger, Auth and Connectivity commands, connection profiles and authentication
will be implemented in subsequent changes. Users of existing v3 scripts must
keep a compatible v3 binary until the commands they need are available in v4.

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
