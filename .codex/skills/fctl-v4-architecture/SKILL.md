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
- `todos/01-v4-isolated-skeleton.md`

Read ADRs as needed:

- `docs/adr/0001-contexts-as-primary-target.md`
- `docs/adr/0002-auth-is-decoupled-from-cloud.md`
- `docs/adr/0003-api-version-resolution.md`
- `docs/adr/0004-cobra-thin-runtime.md`
- `docs/adr/0005-build-v4-in-isolated-directory.md`

## Core Rules

- Do not make Formance Cloud membership required for stack commands.
- Treat contexts as the primary target selector.
- Keep auth as a target-local strategy.
- Use `/versions` plus the compatibility manifest to infer supported API namespaces.
- Commands express product intent; they must not expose API versions as the primary UX.
- Keep CLI flags canonical and product-oriented; map them to version-specific SDK request fields internally.
- Keep Cobra thin. Runtime concerns belong in typed internal packages.
- Build the rewrite under `v4/` until the explicit cutover goal.
- Follow `todos/*.md` in order unless the user explicitly reprioritizes.
- Commit after each logical step.

## CLI Rendering Policy

- Human `plain` output must go through shared rendering helpers instead of writing raw strings directly to `cmd.OutOrStdout()`.
- Use `styledSuccessLine` for successful mutations, `writeStyledKeyValues` for detail views, `writeStyledRows`/`v4render.Table` for lists, `styledEmptyLine` for empty states, and `styledInfoLine` for supporting metadata such as API versions.
- Keep scriptability intact: JSON/YAML output must stay unstyled, non-TTY plain output must remain stable, and ANSI styling must only be emitted for terminal output.
- Direct writes to `cmd.OutOrStdout()` are allowed only for raw payloads such as archives, manifests, logs, or compatibility-preserving fallback paths that are explicitly tracked by tests.
- When adding or touching a command renderer, remove it from the raw-output baseline in `v4/cmd/rendering_policy_test.go` and route its output through the shared helpers.

## Implementation Shape

Prefer this package split under `v4/` during the transition:

```text
v4/cmd/                  Cobra declarations only
v4/internal/runtime/     target resolution, auth, versions, API selection
v4/internal/config/      contexts, defaults, XDG paths, migrations
v4/internal/credentials/ keyring and insecure fallback
v4/internal/capabilities generated manifest and compatibility ranges
v4/internal/commands/    typed product command implementations
v4/internal/render/      table, json, yaml, markdown
v4/internal/prompt/      optional interactive flows
```

## Validation

For command behavior, prefer integration-style tests that execute real CLI commands and assert stdout, stderr, exit codes, and config files. Keep scriptability and non-interactive usage as first-class requirements.
