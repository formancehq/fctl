# ADR 0004: Keep Cobra Thin

Status: Accepted for v4 planning

## Context

Cobra is widely used and already present in `fctl`, but current command implementations mix parsing, auth, client construction, API selection, and rendering.

## Decision

Keep Cobra for routing, flags, help, aliases, deprecations, and shell completions. Move target resolution, authentication, API versioning, and business logic into typed internal packages.

## Consequences

- Existing Cobra knowledge remains useful.
- Command files become smaller and easier to test.
- The runtime can be tested independently from Cobra.
- The CLI can keep a stable user experience while API versions evolve.
