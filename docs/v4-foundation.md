# V4 foundation

The repository now builds the Go module `github.com/formancehq/fctl/v4`.
The executable remains named `fctl`. This change prepares the next major;
it does not create a tag, GitHub release or Homebrew publication.

## Command boundary

`main.go` delegates to `cmd.Execute`. Each execution constructs a fresh Cobra
command tree. Errors propagate to the entrypoint, which writes them to stderr
and exits with status 1. Signal cancellation is propagated through the command
context.

The root registers `version`, `profiles` management and login/logout. Cloud,
Ledger and Connectivity have embedded providers. Auth uses only an external
plugin; Ledger can also use one. The core builds their Cobra commands from
public SDK manifests. Cobra provides help, `--version` and shell completion
commands. Help and completion perform no network access. A fresh Auth target
has only a placeholder with sync/install guidance; a prepared target supplies
full offline metadata. Select the same target to inspect its cached commands.
See [plugin distribution](plugin-distribution.md).

Cloud control-plane operations live under `cloud`; stack administration lives
under `cloud stack`. Host utilities add the authenticated proxy, stdio MCP
bridge and stack token helper. Experimental Apps resolve their own Deploy
audience. See [Cloud management](cloud.md).

The host selects table output on terminals and JSON elsewhere. Presentation is
separate from plugin execution; [output options](output.md) also cover colors
and HTTP diagnostics.

The `version` command prints the CLI version, commit and build date.
The root `--version` flag prints the CLI version. Development defaults are
`v4.0.0-dev`, `-` for the commit and `-` for the build date. GoReleaser injects
these variables under `github.com/formancehq/fctl/v4/cmd/version`.

## Dependencies and generation

The command layer uses upstream Cobra. The previous Formance Cobra fork is
not required by this foundation. The aggregate SDK, go-libs, terminal UI,
Membership and deployment clients have been removed from the dependency graph.
The Auth executable owns its commands and uses the generated service client.
The Auth product plugin module is not a compile-time dependency of fctl, and
no `plugins/auth` adapter is registered. Ledger retains an embedded fallback for its release/v3.0 HTTP
contract. The connection layer handles OAuth2 and verifies Cloud identities
with the coreos OIDC library.

`just pre-commit` runs module tidy, Go generation and lint. `just tests` runs
the race-enabled test suite. Shell completions are generated from the current
command tree, so archives do not advertise removed v3 commands.

## Migration

V4 replaces the v3 command tree and profile format. Cloud management, stack
administration, proxy, MCP and token helpers are implemented with the v4
connection boundary. Terminal forms and searchable selections replace missing-input errors for
interactive callers; scripts can disable them with `--no-input`. No v3
configuration file is read or migrated.

Move old `fctl stack ...` calls to `fctl cloud stack ...`. Stack-scoped Auth and
Ledger commands remain under `auth` and `ledger`; select their Cloud target with
`--organization` and `--stack`. Use `cloud stack --help` to check changed arguments,
confirmation flags and request bodies. See [migration examples](cloud.md#migration-from-v3).

Prepare an exact-version Auth plugin with `plugins sync --service auth` or
`plugins install --service auth --binary PATH` before relying on its full help
or completion. Automatic discovery before execution requires a matching
official release; an unavailable catalogue or missing release is an error.
Local installation remains available before official publication.

Existing automation must adapt to the [v4 module commands](modules.md) and
[profile settings](profiles.md), or keep using a compatible v3 binary.
Connectivity is embedded; its API contract and commands are documented in [Connectivity](connectivity.md).

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
