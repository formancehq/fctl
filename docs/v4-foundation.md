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

## Homebrew release migration

GoReleaser generates `Casks/fctl.rb` and `Casks/fctl@<major>.rb` in
`formancehq/homebrew-tap`. Both casks include macOS and Linux archives for
amd64 and arm64, and the Bash, Zsh and Fish completions. Downloads use
`brew.formance.com`; the previous formula-specific GitHub mirrors are removed.
The macOS post-install hook removes quarantine from the unsigned CLI binary.

Before publishing the first v4 cask, coordinate the tap change with its release
PR: remove `Formula/fctl.rb` and add `"fctl": "fctl"` to the tap's root
`tap_migrations.json`, preserving any existing entries. This lets Homebrew
migrate existing formula installations to the cask. Retain the versioned v3
formula for users who still need v3 commands. New installations can explicitly
select `brew install --cask formancehq/tap/fctl` or
`brew install --cask formancehq/tap/fctl@4`.

The tap migration is a separate release step; this repository does not modify
or publish the tap.
