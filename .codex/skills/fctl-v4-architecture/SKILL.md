---
name: fctl-v4-architecture
description: Use when working on the Formance fctl v4 CLI architecture, contexts, authentication, API version resolution, compatibility manifests, command design, or migrations from fctl v3. This skill keeps future work aligned with the repository's v4 RFC and ADRs.
---

# fctl v4 Architecture Skill

Use this skill for any `fctl` v4 design or implementation work.

## Required Reading

Before changing v4 architecture or commands, read these repository files:

- `docs/rfcs/0001-fctl-v4-architecture.md`
- `docs/cli-v4/command-design.md`
- `docs/cli-v4/compatibility-manifest.md`

Read ADRs as needed:

- `docs/adr/0001-contexts-as-primary-target.md`
- `docs/adr/0002-auth-is-decoupled-from-cloud.md`
- `docs/adr/0003-api-version-resolution.md`
- `docs/adr/0004-cobra-thin-runtime.md`

## Core Rules

- Do not make Formance Cloud membership required for stack commands.
- Treat contexts as the primary target selector.
- Keep auth as a target-local strategy.
- Use `/versions` plus the compatibility manifest to infer supported API namespaces.
- Commands express product intent; they must not expose API versions as the primary UX.
- Keep CLI flags canonical and product-oriented; map them to version-specific SDK request fields internally.
- Keep Cobra thin. Runtime concerns belong in typed internal packages.

## Implementation Shape

Prefer this package split:

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

## Validation

For command behavior, prefer integration-style tests that execute real CLI commands and assert stdout, stderr, exit codes, and config files. Keep scriptability and non-interactive usage as first-class requirements.
