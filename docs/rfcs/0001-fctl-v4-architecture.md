# RFC 0001: fctl v4 Architecture

Status: Draft

## Summary

`fctl` v4 should be rebuilt around Formance targets rather than Formance Cloud membership. The CLI should work naturally against Formance Cloud, self-hosted stacks, and local development stacks. Authentication, target selection, API version selection, and rendering should be independent runtime concerns.

The current CLI is Cloud-first: profiles store a membership URL and root Cloud tokens, and stack clients are created through membership-derived stack access. That makes local and self-hosted usage awkward or impossible without a Cloud membership. It also makes API version selection a command-level concern, so Ledger commands often call the oldest API namespace even when newer SDK namespaces exist.

## Goals

- Make local and self-hosted stack usage first-class.
- Keep Cloud workflows supported without making Cloud the root abstraction.
- Introduce contexts as the primary user-facing target selector.
- Support multiple authentication methods: Cloud device flow, generic OIDC, client credentials, token, and explicit no-auth development mode.
- Select the best compatible API namespace automatically using `/versions` and a generated manifest.
- Keep CLI commands stable at the product language level, even when OpenAPI names change.
- Preserve scriptability with stable JSON/YAML output, non-interactive modes, and predictable exit codes.

## Non-Goals

- Do not expose API namespaces directly in the primary UX, such as `fctl ledger v2 ...`.
- Do not require a new server-side capabilities endpoint for v4 MVP.
- Do not rewrite the public Go SDK as part of the CLI rewrite.
- Do not make interactive TUI flows mandatory.

## Current Problems

- Authentication is structurally tied to Formance Cloud membership.
- The profile model conflates identity, Cloud organization, stack selection, and token storage.
- Stack commands must authenticate through membership before building stack clients.
- Commands call SDK namespaces directly, for example `Ledger.V1` or `Ledger.V2`, making version policy scattered.
- CLI flags mirror API shapes too closely, which makes API evolution leak into the user experience.
- Secrets are stored in profile files rather than being consistently delegated to secure storage.

## Target Model

The v4 runtime should separate these concepts:

- **Context**: named user selection, similar to Docker or Kubernetes contexts.
- **Target**: the actual thing a command talks to, such as a Cloud control plane or a stack data plane.
- **Auth**: how credentials are obtained for that target.
- **Capabilities**: what the CLI infers the target can support.
- **API version policy**: how the CLI chooses among SDK namespaces.
- **Rendering**: how typed command results become tables, JSON, YAML, or human text.

Example context:

```yaml
currentContext: local
contexts:
  local:
    kind: stack
    stackURL: http://localhost/api
    auth:
      method: client_credentials
      issuerURL: http://localhost/api/auth
      clientID: testing
      secretRef: keyring://formance/local/testing
    defaults:
      ledger: default
    api:
      ledger: latest-compatible

  cloud-prod:
    kind: cloud-stack
    cloudURL: https://app.formance.cloud/api
    organization: org_x
    stack: stack_y
    auth:
      method: cloud_device
      account: user@example.com
    api:
      ledger: latest-compatible
```

## Command Model

Commands represent product intent, not OpenAPI operations. For example:

```bash
fctl ledger transactions list
fctl ledger transactions revert <id>
fctl ledger schemas insert <version>
```

Each command parses a canonical input model, then delegates to a versioned handler selected by the runtime.

```go
type VersionedCommand[In any, Out any] struct {
    Product  string
    Feature  string
    Handlers []VersionedHandler[In, Out]
}

type VersionedHandler[In any, Out any] struct {
    APIVersion APIVersion
    Run        func(context.Context, *formance.Formance, In) (Out, error)
}
```

The Cobra command should only parse flags, construct the canonical input, and call the typed command package.

## API Version Selection

The server currently exposes `/versions`, not a full capabilities endpoint. That is sufficient for v4.

Runtime flow:

1. Call `sdk.GetVersions(ctx)`.
2. Read component versions, such as `ledger=2.3.4`.
3. Map component versions to supported API namespaces through a small compatibility table.
4. Intersect server-supported API namespaces with SDK handlers available in the CLI.
5. Choose the highest compatible version by default.

Example:

- CLI has handlers for `ledger.v1`, `ledger.v2`, `ledger.v3`.
- `/versions` reports Ledger `2.3.4`.
- Compatibility table says Ledger `>=2.0.0 <3.0.0` supports `v1` and `v2`.
- The command uses `Ledger.V2`.

Users can still force a version:

```bash
fctl ledger transactions list --api-version v1
fctl ledger transactions list --latest
```

The default should be `latest-compatible`.

## Compatibility Manifest

Most operation metadata should be generated from the released OpenAPI document, for example:

`https://github.com/formancehq/stack/releases/download/v3.2.4/generate.json`

The OpenAPI tags already contain names like `ledger.v1`, `ledger.v2`, `payments.v1`, `payments.v3`, `orchestration.v1`, and `orchestration.v2`. The generator can produce a manifest mapping product, API namespace, operation ID, and path.

The only manual compatibility data should be component-version ranges to API namespaces, such as:

```go
ledger >= 1.0.0 < 2.0.0 => v1
ledger >= 2.0.0 < 3.0.0 => v1, v2
ledger >= 3.0.0          => v1, v2, v3
```

## Flag and Parameter Design

CLI flags should use Formance product vocabulary, not generated API field names. If an API changes `account` to `address`, but the product concept is still an account address, the CLI should expose one canonical flag and map it internally.

Rules:

- If only the API name changed, keep one canonical CLI flag.
- If an old CLI flag is widely used, keep it as a deprecated alias.
- If a flag only works on newer API versions, keep it visible and validate it against runtime capabilities.
- If concepts diverge semantically, create separate product-level commands rather than version-suffixed commands.

## Technical Stack

- Keep Cobra and pflag for command routing, help, aliases, deprecations, and shell completions.
- Keep Cobra thin; do not put target/auth/version logic in command files.
- Use Charmbracelet Huh for optional interactive setup flows.
- Use Lip Gloss and Glamour for terminal and Markdown rendering where useful.
- Use a system keyring for secrets and explicit insecure fallback only when requested.
- Use XDG-aware paths for config, cache, and state.
- Use `testscript` style integration tests for real CLI behavior.
- Use GoReleaser for packaging, checksums, completions, and package manager artifacts.

## Proposed Package Shape

```text
cmd/                  Cobra declarations only
internal/runtime/     target resolution, auth, versions, API selection
internal/config/      contexts, defaults, XDG paths, migrations
internal/credentials/ keyring and insecure fallback
internal/capabilities generated manifest and compatibility ranges
internal/commands/    typed product command implementations
internal/render/      table, json, yaml, markdown
internal/prompt/      optional interactive flows
```

## Migration

The v4 CLI should import v3 profiles into contexts:

- A v3 Cloud profile becomes a `cloud-stack` or `cloud` context.
- Membership tokens move out of profile files into the credential store when possible.
- Existing default organization and stack become context defaults.
- The v3 config should not be deleted automatically.

## Open Questions

- Exact naming for `context` versus `target`.
- Whether `fctl transaction list` aliases should exist beside `fctl ledger transactions list`.
- Whether compatibility ranges should live in the CLI repo, the SDK repo, or both.
- How aggressively to warn when a newer API namespace is available.
