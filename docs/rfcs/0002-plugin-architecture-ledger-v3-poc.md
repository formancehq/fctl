# RFC 0002: Plugin Architecture — Ledger v3 as Proof of Concept

Status: Draft

## Context

PR #153 proposes a monolithic v4 rewrite where all product commands live in a single
binary with multi-version handlers. PR #126 proposes a plugin architecture where each
product ships as an independent binary. Both approaches have been reviewed and
discussed.

The consensus is that the plugin model solves real structural problems (version bloat,
intra-version capability gaps, `--help` superset, product team ownership), but needs
validation on a real case before committing to it as the target architecture.

This RFC uses **Ledger v3** as the concrete proof of concept. Every scenario is
described with exact CLI output and terminal sessions, not abstract flows.

## Goals

- Validate the plugin architecture through a complete Ledger v3 user journey.
- Answer every operational question raised during the #153 review.
- Determine whether plugins should become the target architecture for fctl v4.
- Keep the plugin infrastructure scope minimal — only what Ledger v3 needs.

## Non-Goals

- Migrate all products to plugins in this RFC.
- Change the auth/profile model (reuse v4 profiles as-is).
- Design a plugin marketplace or community plugin ecosystem.

---

## Plugin Location Convention

Each plugin lives **in the product's own repository**, under a normalized path:

```
<product-repo>/
  cmd/fctl-plugin/
    main.go          # entry point: pluginsdk.Serve(...)
    go.mod           # separate Go module, depends on pluginsdk
    ...
```

For example:

| Product | Repository | Plugin path |
|---|---|---|
| Ledger | `formancehq/ledger` | `cmd/fctl-plugin/` |
| Payments | `formancehq/payments` | `cmd/fctl-plugin/` |
| Orchestration | `formancehq/orchestration` | `cmd/fctl-plugin/` |

This is a deliberate choice:

- **Ownership is clear.** The plugin is maintained by the product team, next to the
  product code. No cross-repo PRs needed to add a flag or fix a bug.
- **Versioning is natural.** The plugin is tagged and released alongside the product.
  When the Ledger team releases v3.2.0, the plugin binary for v3.2.0 is built from
  the same commit.
- **Protos stay local.** The plugin imports the product's own proto definitions
  directly (e.g., `ledger/proto/...`). No shared proto repository, no SDK
  indirection.
- **CI is the product's CI.** The plugin binary is built and published by the same
  pipeline that builds the product. No separate release process to maintain.

The `cmd/fctl-plugin/` directory is a separate Go module (`go.mod`) that depends on
the `pluginsdk` package. It does not import fctl core — only the plugin SDK and the
product's own packages.

**Registry binaries point to product releases:**

```yaml
plugins:
  ledger:
    repo: formancehq/ledger            # GitHub repo — binary URLs are derived
    type: stack
    versions:
      3.2.0:                           # = service version (enforced)
        compatibleWith: ">=3.0.0"      # backward compat range
      3.3.0:
        compatibleWith: ">=3.3.0"      # breaking gRPC change at 3.3.0
```

The **plugin version must match the service version**. Since the plugin is built
and released from the same repo at the same commit, there is no reason for them
to diverge. This is enforced, not a convention:

- The plugin binary is built from the service's release tag (e.g., `v3.2.0`).
- `GetManifest()` returns the same version as the service.
- The registry entry key matches the release tag.

This eliminates a class of confusion ("which plugin version do I need for
service version X?"). The answer is always the same version.

The `compatibleWith` range exists to express **backward compatibility within a
plugin version** — a plugin built at v3.2.0 may work with any service from
v3.0.0 onward if the gRPC API is stable across that range.

**Registry metadata vs artifact distribution are separate concerns.**

The registry tracks **what exists and what is compatible** (metadata). How and
where binaries are downloaded from is the **distribution backend** — a separate
layer that the registry points to but does not define.

```yaml
plugins:
  ledger:
    repo: formancehq/ledger
    type: stack
    distribution: github-releases     # distribution backend for this plugin
    versions:
      3.2.0:
        compatibleWith: ">=3.0.0"
```

**Distribution backends:**

| Backend | URL pattern | Auth | Use case |
|---|---|---|---|
| `github-releases` (default) | `https://github.com/{repo}/releases/download/v{version}/fctl-plugin-{name}-{os}-{arch}` | None (public) or GitHub token (private) | Public repos, POC |
| `private-releases` | Same pattern, but requires a GitHub token with `repo` scope | GitHub PAT or GITHUB_TOKEN | Private product repos |
| `oci` | `oci://{registry}/{name}:{version}` | Registry auth (Docker config) | Air-gapped, enterprise, custom registries |
| `url` | Explicit URL per version (fallback) | Custom | Anything else |

For the **Ledger v3 POC**, `github-releases` is sufficient — the ledger repo is
public. But the registry format must support other backends from the start so
that private or enterprise products don't require a redesign.

When `distribution` is `private-releases`, fctl uses the same URL convention but
includes a GitHub token from the user's environment (`GITHUB_TOKEN`) or from the
credential store. This covers the case where a product repo is private but the
plugin needs to be distributed to authorized users.

**Checksums** are published alongside the binaries in the GitHub release (standard
`checksums.txt` artifact), and fctl verifies them after download regardless of
the distribution backend.

The registry stays minimal — metadata only, no binary URLs or checksums inline.

---

## Built-in vs Plugin Resolution

Ledger v1/v2 commands **remain built-in** in fctl. The plugin system activates
for Ledger v3+. But the mechanism is not tied to major version boundaries — it
works at **any semver granularity**.

The `compatibleWith` field in the registry is a **service version** semver
range — not a stack version. It matches against the version reported by
`/versions` for the corresponding service (e.g., `ledger: "3.2.0"`), or by
`GetRegionVersions()` in Cloud mode.

A plugin version can target a major, a minor, or even a patch range:

```yaml
plugins:
  ledger:
    versions:
      3.2.0:
        compatibleWith: ">=3.0.0 <3.3.0"    # covers Ledger 3.0.x through 3.2.x
      3.3.0:
        compatibleWith: ">=3.3.0 <4.0.0"    # new features from Ledger 3.3.0
      4.0.0:
        compatibleWith: ">=4.0.0"            # breaking change in Ledger 4.x
```

This means if Ledger 3.3.0 adds a new gRPC endpoint or changes a response
shape, a new plugin version can be published to match it — without waiting for
a major version bump.

Each plugin maps to exactly one service. The plugin name matches the service
name reported by `/versions`:

| `/versions` key | Plugin name | `compatibleWith` matches against |
|---|---|---|
| `ledger: "3.2.0"` | `ledger` | Ledger service version |
| `payments: "3.1.0"` | `payments` | Payments service version |
| `orchestration: "2.0.0"` | `orchestration` | Orchestration service version |

**Resolution order** when a user runs `fctl ledger transactions list`:

1. fctl detects the stack's Ledger version (via membership API or `/versions`).
2. If a **plugin** with a matching `compatibleWith` is installed → use it.
3. If no plugin matches but the version is covered by **built-in** commands
   (< 3.0.0 for Ledger) → use built-in.
4. If neither → trigger auto-discovery from the registry.

```
plugin with matching compatibleWith installed?
  ├── yes → use plugin
  └── no
       built-in covers this version?
         ├── yes → use built-in
         └── no  → auto-discover + prompt
```

Plugins take precedence over built-in commands when both could match. This
allows a product team to ship a plugin that replaces the built-in at any
version boundary, not just major ones.

This means:

- **Zero disruption for existing users.** Ledger v2 stacks work exactly as before.
  No plugin to install, no behavior change.
- **Plugins are opt-in by necessity.** A user only encounters the plugin system when
  their stack runs a version that requires it.
- **The built-in commands can be frozen.** Once Ledger v2 is stable, no new handlers
  or feature-gating code needs to be added to fctl for Ledger. New Ledger features
  land exclusively in the plugin.
- **Minor version differences are handled.** A new feature introduced in Ledger
  3.3.0 gets its own plugin version. Users on 3.2.x keep the old plugin. No
  feature-gating `if` blocks, no intra-version capability gaps.
- **Gradual migration path.** Other products (payments, wallets, etc.) can follow
  the same pattern at whatever version boundary makes sense — major, minor, or
  even when they want to switch from REST to gRPC. Until then, their commands
  stay built-in.

---

## User Journeys

### 1. Cloud user gets the right ledger plugin

Marie uses Formance Cloud. She has fctl configured with a cloud profile.

```
$ fctl profile show
Profile:       default
Kind:          cloud-stack
Cloud URL:     https://app.formance.cloud/api
Organization:  org_acme
Stack:         stack_prod
```

She runs a ledger command:

```
$ fctl ledger transactions list
```

fctl detects that the stack runs Ledger v3.1.0 (>= 3.0.0). The built-in ledger
commands only support v1/v2, so fctl checks for a plugin. No plugin is installed
yet, so it triggers auto-discovery.

**Auto-discovery flow:**

1. fctl reads the active profile — it's a `cloud-stack` profile with org and stack.
2. fctl gets the Ledger version (membership API → `GetRegionVersions`) → `3.1.0`.
3. Resolution: no plugin installed, built-in doesn't cover v3.x → auto-discover.
4. fctl fetches the plugin registry.
5. fctl finds `fctl-plugin-ledger` with a `compatibleWith` matching `3.1.0`.
6. fctl prompts:

```
$ fctl ledger transactions list

  The "ledger" command requires a plugin that is not installed.

  Stack "stack_prod" runs Ledger v3.1.0.
  Compatible plugin: fctl-plugin-ledger v3.2.0

  Install it now? [Y/n] y

  Downloading fctl-plugin-ledger v3.2.0 (darwin/arm64)... done
  Verifying checksum... ok
  Plugin installed.

  ID          TIMESTAMP            REFERENCE   POSTINGS
  42          2026-05-14T10:00:00Z ref-001     2
  41          2026-05-14T09:45:00Z             1
  ...
```

On subsequent runs, the plugin is already installed. No prompt, no delay beyond the
normal gRPC plugin handshake.

**Non-interactive mode:**

```
$ fctl --non-interactive ledger transactions list
Error: stack "stack_prod" runs Ledger v3.1.0, which requires
       the "ledger" plugin. No compatible version is installed.

  Run: fctl plugin install ledger

$ echo $?
1
```

No implicit install in non-interactive mode. The error message includes the exact
command to fix it.

### 2. User switches stack or targets another version

Marie has the Ledger v3 plugin installed for her prod stack. She switches to a
staging stack that runs Ledger v2.

**Switching to a Ledger v2 stack — built-in kicks in:**

```
$ fctl profile use staging
Switched to profile "staging".

$ fctl ledger transactions list

  ID     TIMESTAMP            REFERENCE
  18     2026-05-13T14:00:00Z
  17     2026-05-13T13:30:00Z ref-staging-1
  ...
```

No prompt, no plugin involved. fctl detects that `stack_staging` runs Ledger
v2.8.0 (< 3.0.0), so it uses the **built-in** ledger commands. The installed
v3 plugin is ignored.

**Switching back to Ledger v3 — plugin kicks in:**

```
$ fctl profile use default
Switched to profile "default".

$ fctl ledger transactions list

  ID     TIMESTAMP            REFERENCE   POSTINGS
  42     2026-05-14T10:00:00Z ref-001     2
  ...
```

fctl detects Ledger v3.1.0, finds the already-installed plugin, uses it.
No download, no prompt.

**How it works under the hood:**

Same resolution order as described above:

1. fctl resolves the active profile and gets the Ledger component version.
2. Check for a plugin with matching `compatibleWith` → no plugin covers v2.8.0.
3. Check built-in commands → built-in covers v2.8.0. Use it.

When switching back to the prod profile (Ledger v3.1.0):

1. Check for a plugin → `fctl-plugin-ledger v3.2.0` matches. Use it.

The switch between built-in and plugin is **transparent to the user**. The
command is the same (`fctl ledger transactions list`), only the execution
path changes based on the stack version.

### 3. Self-hosted user

Thomas runs a self-hosted Formance stack. He sets up fctl with a direct stack profile.

```
$ fctl login --target open-source --stack-url https://formance.internal/api
Profile "default" created (kind: stack).

$ fctl ledger transactions list

  The "ledger" command requires a plugin that is not installed.

  Detecting stack capabilities...
  GET https://formance.internal/api/versions → Ledger v3.0.2

  Compatible plugin: fctl-plugin-ledger v3.2.0

  Install it now? [Y/n] y

  Downloading fctl-plugin-ledger v3.2.0 (linux/amd64)... done
  Plugin installed.

  ID     TIMESTAMP            REFERENCE   POSTINGS
  ...
```

**Difference from Cloud:** Instead of calling the membership API, fctl calls the
stack's `/versions` endpoint directly to discover component versions. The rest of
the flow (registry lookup, download, install) is identical.

**Air-gapped environments:**

For environments without internet access, plugins can be installed from a local path:

```
$ fctl plugin install ledger --path /mnt/artifacts/fctl-plugin-ledger
Plugin "ledger" installed from local path.
```

Or pre-bundled in a Docker image / system package alongside fctl.

### 4. Local developer

Léa is working on the Ledger. The plugin source lives in the ledger repo under
`cmd/fctl-plugin/`.

**Building and installing from source:**

```
$ cd ~/code/ledger
$ fctl plugin install ledger --path ./cmd/fctl-plugin

  Detected Go module. Building...
  go build -o ~/.config/formance/fctl/plugins/ledger/dev/fctl-plugin-ledger .
  Fetching manifest... ok
  Plugin "ledger" installed (version: dev).
```

fctl detects the `go.mod` in the directory, runs `go build`, copies the binary, and
fetches the manifest. The version is tagged as `dev`.

Since the plugin is a sub-module of the ledger repo, it imports the ledger's own
proto definitions and internal packages directly. Changes to the ledger's gRPC API
and the plugin's CLI commands happen in the same commit.

**Development loop:**

After making changes to the plugin or the ledger:

```
$ fctl plugin install ledger --path ./cmd/fctl-plugin   # rebuilds
$ fctl ledger transactions list                          # test immediately
```

Or with a pre-built binary:

```
$ cd ~/code/ledger
$ go build -o ./fctl-plugin-ledger ./cmd/fctl-plugin
$ fctl plugin install ledger --path ./fctl-plugin-ledger
```

**Running without install (ephemeral):**

For quick testing without touching the plugin config:

```
$ fctl --plugin-binary ledger=./fctl-plugin-ledger ledger transactions list
```

This loads the plugin for this invocation only, without modifying `plugins.json`.

**Testing against a local stack:**

```
$ fctl profile use local
$ fctl ledger transactions list --ledger default

  ID     TIMESTAMP                 REFERENCE   POSTINGS
  3      2026-05-15T09:00:00Z      test-ref    1
  2      2026-05-15T08:45:00Z                  2
  1      2026-05-15T08:30:00Z      init        1
```

The plugin receives the stack URL and auth token from fctl via `AuthContext`. The
developer doesn't need to configure the plugin separately.

### 5. Debugging through a plugin

Marie hits an issue and wants to debug.

**`--debug` flag:**

```
$ fctl --debug ledger transactions list

  [fctl]    Profile: default (cloud-stack)
  [fctl]    Stack: stack_prod → Ledger v3.1.0
  [fctl]    Plugin: fctl-plugin-ledger v3.2.0
  [fctl]    Spawning plugin process...
  [fctl]    gRPC handshake: ok (12ms)
  [fctl]    → Execute(path: "transactions/list", flags: {ledger: "default"})
  [plugin]  Connecting to grpc.stack-prod.formance.cloud:443
  [plugin]  → BucketService.ListTransactions(ledger: "default", page_size: 10)
  [plugin]  ← 10 transactions (23ms)
  [fctl]    ← ExecuteResponse.Success (json_data: 847 bytes)
  [fctl]    Rendering table from display schema...

  ID     TIMESTAMP            REFERENCE   POSTINGS
  42     2026-05-14T10:00:00Z ref-001     2
  ...
```

**How it works:**

1. fctl sets `debug: true` in the `AuthContext` sent to the plugin.
2. The plugin checks `req.AuthContext.Debug` and writes diagnostics to stderr.
3. fctl captures the plugin's stderr and prefixes it with `[plugin]`.
4. fctl's own debug output is prefixed with `[fctl]`.

Both streams go to the user's stderr, interleaved chronologically. The plugin's
stdout is reserved for the gRPC protocol (managed by go-plugin).

**gRPC-level debugging:**

For deeper issues, the `GRPC_GO_LOG_SEVERITY_LEVEL` environment variable works
on both the fctl side (plugin client) and the plugin side (server connecting to
the stack):

```
$ GRPC_GO_LOG_SEVERITY_LEVEL=info fctl --debug ledger transactions list
```

**Plugin crash:**

If a plugin process crashes, fctl catches the error and reports it clearly:

```
$ fctl ledger transactions list

  Error: plugin "ledger" crashed during execution.

  Exit code: 2
  Stderr:
    panic: runtime error: index out of range [0] with length 0

    goroutine 1 [running]:
    main.handleTransactionsList(...)
        /code/fctl-plugin-ledger/handle_transactions.go:45

  This is a bug in the ledger plugin (v3.2.0), not in fctl.
  Report: https://github.com/formancehq/ledger/issues
```

fctl distinguishes between its own errors and plugin errors, and points the user
to the right repository.

### 6. Plugin missing or needs update

When a plugin is needed and not installed, fctl always attempts auto-discovery
first (see journey #1). The user only sees an error if auto-discovery is not
possible or was declined.

**No profile active (no stack to discover from):**

```
$ fctl --profile "" ledger transactions list
Error: command "ledger" requires a plugin, but no profile is active
       so the required version cannot be determined.

  Configure a profile first: fctl login
```

**Non-interactive mode (auto-install is disabled):**

```
$ fctl --non-interactive ledger transactions list
Error: stack "stack_prod" runs Ledger v3.1.0, which requires
       the "ledger" plugin. No compatible version is installed.

  Run: fctl plugin install ledger

$ echo $?
1
```

Note: if the stack runs Ledger v2.x, the built-in commands handle it. The
plugin system is never involved.

**Plugin updates are driven by service upgrades.**

There is no `fctl plugin update` command. Since plugin version = service version,
a plugin update only happens when the service version changes on the stack. When
it does, fctl detects the mismatch through the normal resolution flow and
auto-discovers the new plugin version — same as a first install.

This means plugin lifecycle is fully automatic: install, upgrade, and version
selection are all handled by auto-discovery. The user never manages plugin
versions manually (except in air-gapped or development scenarios via `--path`).

### 7. Version retirement

The Ledger team releases Ledger v5.0.0. Ledger v3.x reaches end-of-life.

**Registry-side:**

The registry maintainer removes v3.x plugin entries from `registry.yaml`:

```yaml
plugins:
  ledger:
    repo: formancehq/ledger
    type: stack
    versions:
      # v3.x entries removed — no longer supported
      4.0.0:
        compatibleWith: ">=4.0.0 <5.0.0"
      5.0.0:
        compatibleWith: ">=5.0.0"
```

**Impact on existing users:**

- Users with `fctl-plugin-ledger v3.2.0` already installed: **nothing breaks**.
  The binary is on disk, it continues to work against their Ledger v3.x stacks.
- `fctl plugin install ledger` on a Ledger v3.x stack: no compatible version
  found in registry. fctl prints:

```
  Error: no compatible plugin version found for Ledger v3.2.0.
         Ledger v3.x has reached end-of-life.

  Options:
    - Upgrade your stack to Ledger v4.x or later
    - Install a specific version manually:
      fctl plugin install ledger --path /path/to/binary
```

**Deprecation notice before removal:**

Before removing v3.x from the registry, the registry can mark it as deprecated:

```yaml
      3.2.0:
        compatibleWith: ">=3.0.0 <4.0.0"
        deprecated: "Ledger v3.x reaches end-of-life on 2027-03-01"
```

fctl shows this on install and on each execution:

```
$ fctl ledger transactions list
  Warning: plugin "ledger" v3.2.0 is deprecated.
           Ledger v3.x reaches end-of-life on 2027-03-01.
           Consider upgrading your stack.

  ID     TIMESTAMP            ...
```

**What about the built-in Ledger v2 commands?**

Ledger v2 built-in commands are retired through a normal fctl release cycle.
When fctl drops support for Ledger v2, the built-in handlers are removed in a
new fctl major version. Users on Ledger v2 stay on the older fctl version.

---

## Plugin Protocol Additions

The existing protocol from #126 is sufficient for the Ledger v3 POC with two
additions:

### 1. Stderr forwarding for debug output

The current go-plugin framework already captures plugin stderr. fctl should
prefix plugin stderr lines with `[plugin]` and forward them when `--debug` is
active. No protocol change needed — this is a fctl-side behavior.

### 2. Multi-version plugin storage

As products release new major versions over time, multiple plugin versions may
coexist. `plugins.json` supports this:

```json
{
  "plugins": [
    {
      "name": "ledger",
      "versions": {
        "3.2.0": {
          "compatibleWith": ">=3.0.0 <4.0.0",
          "path": ""
        },
        "4.0.0": {
          "compatibleWith": ">=4.0.0",
          "path": ""
        }
      }
    }
  ]
}
```

At runtime, fctl follows the standard resolution order: plugin first, then
built-in fallback, then auto-discovery.

---

## Rendering

Plugins should not render output themselves. The `ExecuteResponse` should return
structured data, and fctl core handles all rendering:

```protobuf
message ExecuteSuccess {
    string json_data = 1;       // structured output (always populated)
    // rendered_text removed — fctl core renders from json_data
}
```

fctl core provides rendering based on `--output`:
- `plain`: styled table from structured data (using shared renderers)
- `json`: pretty-printed JSON
- `yaml`: YAML output

This ensures consistent styling, colors, table formatting, and `--no-color`
behavior across all plugins without each plugin reimplementing rendering.

To support this, the plugin manifest declares a **display schema** per command:

```protobuf
message CommandSpec {
    // ... existing fields ...
    DisplaySchema display = 14;
}

message DisplaySchema {
    repeated ColumnSpec columns = 1;     // for list commands
    repeated SectionSpec sections = 2;   // for show/inspect commands
}

message ColumnSpec {
    string header = 1;      // "ID"
    string json_path = 2;   // "$.id"
    string format = 3;      // "timestamp", "number", etc.
}
```

This keeps rendering logic in fctl while letting plugins declare how their
data should be displayed.

---

## Auth Integration

The plugin system reuses the v4 profile/auth model as-is. fctl core resolves
credentials based on the profile kind and passes them to the plugin via
`AuthContext`:

| Profile kind | Auth resolved by fctl | AuthContext fields populated |
|---|---|---|
| `stack` (local/self-hosted) | Client credentials or none | `service_url`, `access_token`, `issuer_url` |
| `cloud-stack` | Membership → stack token | `service_url`, `access_token`, `membership_url`, `organization_id`, `stack_id` |
| `cloud` | Membership token | `membership_url`, `membership_token`, `organization_id` |

The plugin never handles OAuth flows, token refresh, or profile management.
It receives a ready-to-use token and endpoint.

For Ledger v3 specifically, the plugin uses `AuthContext.service_url` to
connect via gRPC and `AuthContext.access_token` as a bearer token.

---

## Scope for the POC

The POC is limited to:

1. **fctl core** (in `formancehq/fctl`): plugin manager, registry client,
   multi-version storage, auto-discovery (Cloud via membership, self-hosted via
   `/versions`), centralized rendering from structured output, `pluginsdk`
   package.
2. **fctl-plugin-ledger** (in `formancehq/ledger` at `cmd/fctl-plugin/`):
   Ledger v3 gRPC commands, adapted to return structured data instead of
   rendered text. Built and released by the ledger CI pipeline.
3. **Registry** (in `formancehq/fctl-plugin-registry`): a `registry.yaml`
   pointing to plugin binaries hosted on each product's GitHub releases.

Everything else (payments, wallets, flows, etc.) stays built-in until the POC
is validated.

---

## Success Criteria

The POC is considered successful if:

1. A Ledger v2 stack works exactly as before — built-in commands, no plugin
   involved, no behavior change.
2. A Cloud user targeting a Ledger v3 stack can run `fctl ledger transactions list`
   with zero manual plugin management (auto-discovery installs the right version).
3. Switching profiles between a Ledger v2 and v3 stack is seamless — fctl
   transparently uses built-in commands for v2 and the plugin for v3.
4. A self-hosted user can install and use the ledger plugin with one command.
5. A developer can build, install, and test a plugin change in under 30 seconds.
6. `--debug` shows clear, labeled output from both fctl and the plugin.
7. Error messages for missing plugins include actionable commands.
8. `--help` for ledger commands on a v3 stack shows only flags relevant to the
   installed plugin version (not the v2 built-in flags).
9. The plugin binary is small and self-contained — no monolithic SDK dependency.

---

## Open Questions

- Should the registry be a GitHub repo with YAML, or a simple HTTP JSON endpoint?
  YAML is easier to review in PRs. JSON is easier to consume programmatically.
  Both work; the POC can start with either.
- Should fctl bundle a "fallback" set of plugins in its release artifacts for
  offline first-run scenarios?
- What is the minimum fctl core version that the plugin protocol targets? This
  determines whether we can ship the plugin infrastructure as a patch on the
  current v3 fctl or if it requires the v4 foundation.
- The POC uses `github-releases` as the distribution backend (public repo).
  The `private-releases` and `oci` backends are designed in the registry format
  but not implemented in the POC. These should be validated before plugins are
  adopted for products with private repositories.
