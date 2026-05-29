# fctl v4 Runtime Behavior

This page documents the current behavior that is not obvious from command help
alone. It is intended as the operational companion to the architecture RFC and
the command reference.

## Target Model

v4 separates target selection, authentication, API version resolution, and
rendering.

- `profile` selects the configured target.
- `--organization` and `--stack` are Cloud or EE stack overrides.
- Stack data-plane commands can run against a direct `stack` profile without
  Cloud membership.
- Cloud control-plane commands require a `cloud` or `cloud-stack` profile.
- Product commands ask the runtime for the best supported SDK namespace instead
  of exposing API namespaces in command paths.

## Login

`fctl login` is the primary user-facing authentication flow. It creates or
replaces the selected profile, defaulting to `default`.

Visible login modes:

- `fctl login --target cloud`
- `fctl login --target ee --membership-url <url>`
- `fctl login --target open-source --stack-url <url>`
- `fctl login --target cloud --client-id <id> --client-secret-stdin`
- `fctl login --target ee --membership-url <url> --client-id <id> --client-secret-stdin`

Interactive terminals use a Charmbracelet Huh wizard. The wizard asks only for
fields that were not provided by flags, then prints the selected values as
styled confirmation lines. Non-TTY execution and `--non-interactive` keep stable
flag validation and never require interactive prompts.

Browser/device login uses the Cloud or EE OIDC device authorization endpoint.
The CLI opens the browser when possible, prints the verification URL when it
cannot, then polls until authentication succeeds or expires.

Static-token login is not exposed as a login choice. The config model can still
represent token credentials for explicit contexts and migration compatibility,
but users should not see a "static token" option in the login wizard.

## Cloud Scopes

Device login requests the full Cloud organization scope set needed by visible
Cloud commands, including stack, region, policy, user, invitation, OAuth client,
authentication-provider, logs, and feature scopes.

Stored device credentials are validated before use. If the access token is
expired or does not contain the scopes required by the command, the device token
source refreshes credentials with the missing command scopes and stores the
refreshed token set.

Scope errors such as `invalid_scope` or `missing one of scopes` usually mean the
stored credentials predate the current scope list or the Cloud authorization
server rejected a requested scope. Re-run `fctl login` after scope list changes.

## API Version Resolution

Stack product commands resolve API versions at runtime:

1. Fetch `<stack-url>/versions`.
2. Read component versions and health values.
3. Map component versions to supported API namespaces using the generated
   compatibility data.
4. Intersect target support with handlers compiled into the CLI.
5. Select the highest compatible API namespace unless `--api-version` pins one.

The command path stays product-oriented, for example
`ledger transactions list`; generated SDK namespaces remain implementation
details.

The current resolver works at API-namespace granularity. It does not yet model
every feature or parameter difference inside a namespace. See
`docs/cli-v4/versioning-and-ownership.md` for the known gaps and the planned
capabilities improvements.

## Cloud Stack Creation Wait

`cloud stacks create` waits for availability by default. Use `--no-wait` to
return immediately after the Cloud API accepts stack creation.

The wait loop uses two signals in order:

1. It probes `<stack-url>/versions`. The stack is considered available only when
   the endpoint returns HTTP 200, returns at least one version, and every
   returned version has `health=true`.
2. If `/versions` is unavailable or unhealthy, it falls back to the Cloud
   membership stack status and keeps polling until the stack reaches `READY`.

The wait loop polls every two seconds, has a ten-minute default timeout, and
prints a styled spinner in interactive terminals. The spinner message includes
the stack URL when the Cloud API has returned it.

The final human output is a single styled success block containing the stack ID
and URL. It should not print a second unstyled "created" line.

## Output Rendering

Human output uses the shared terminal styling helpers in `v4/cmd/terminal_ui.go`.
Commands should not print raw key/value lines or plain success strings unless
they are intentionally in structured output mode.

Rendering rules:

- Human output is styled and goes to stdout.
- Warnings, debug traces, and deprecation messages go to stderr.
- `--output json` and `--output yaml` bypass decorative human styling.
- `--no-color` disables color but keeps stable spacing and labels.
- Secrets are masked in human output. Use stdin flags such as `--secret-stdin`
  for secret input.
- Prompt cancellation should return a clean cancellation error without leaking
  low-level prompt messages such as `prompt cancelled`.

The rendering policy is enforced by tests that scan command implementations for
unstyled output paths.

## HTTP Debugging

`-d` and `--debug` enable technical diagnostics on stderr.

Debug mode prints:

- selected context, target kind, and target URL;
- outgoing HTTP method and URL;
- request headers, with sensitive headers redacted;
- response status and response headers, with sensitive headers redacted.

Sensitive headers include `Authorization`, `Cookie`, and `Set-Cookie`.

Debug output is for troubleshooting and must not change stdout rendering or
structured output.

## Hidden Product Surface

Some code paths exist but are hidden until the product contract is explicit:

- `cloud apps ...` is hidden from visible help and command audits.
- `ledger transactions explain` is hidden until the public stack spec and
  `formance-sdk-go` expose `explainTransaction`.

Hidden commands can remain covered by tests, but they are not part of the user
contract and should not be listed as visible commands in the command reference.
