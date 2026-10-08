# V4 foundation

The repository now builds the Go module `github.com/formancehq/fctl/v4`.
The executable remains named `fctl`. This change prepares the next major;
it does not create a tag, GitHub release or Homebrew publication.

## Command boundary

`main.go` delegates to `cmd.Execute`. Each execution constructs a fresh Cobra
command tree. Errors propagate to the entrypoint, which writes them to stderr
and exits with status 1. Signal cancellation is propagated through the command
context.

The root registers `version`. Cobra provides help, `--version` and shell
completion commands. No service client, profile, authentication or network
access is required to inspect help or version information.

The `version` command prints the CLI version, commit and build date.
The root `--version` flag prints the CLI version. Development defaults are
`v4.0.0-dev`, `-` for the commit and `-` for the build date. GoReleaser injects
these variables under `github.com/formancehq/fctl/v4/cmd/version`.

## Dependencies and generation

The command layer uses upstream Cobra. The previous Formance Cobra fork is
not required by this foundation. The aggregate SDK, go-libs, terminal UI,
Membership and deployment clients have been removed from the dependency graph.
Service-specific clients will be selected when their modules are implemented.

`just pre-commit` runs module tidy, Go generation and lint. `just tests` runs
the race-enabled test suite. Shell completions are generated from the current
command tree, so archives do not advertise removed v3 commands.

## Migration

The v3 CLI command surface has been removed, including service commands,
Cloud and Stack management, profiles, login, the proxy, MCP and the interactive
prompt. No configuration file is read or migrated by this foundation.

Existing automation must continue to use a compatible v3 binary until its
required commands are implemented in v4. Ledger, Auth, Connectivity, direct
connections and Cloud authentication remain subsequent implementation work.
