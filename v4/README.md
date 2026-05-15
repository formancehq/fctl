# fctl v4

This directory contains the isolated fctl v4 rewrite. The existing repository
root remains the current v3 implementation during the transition.

## Run

From this directory:

```bash
go run . --help
go run . version
```

## Test

From this directory:

```bash
go test ./...
```

## Documentation

The v4 user and architecture documentation lives in the repository-level
`docs/` directory:

- `docs/rfcs/0001-fctl-v4-architecture.md`: target architecture and core
  separation of context, target, auth, capabilities, API version selection, and
  rendering.
- `docs/adr/`: accepted design decisions for contexts, auth, API version
  resolution, Cobra boundaries, and the isolated v4 directory.
- `docs/cli-v4/command-reference.md`: current visible command families.
- `docs/cli-v4/runtime-behavior.md`: operational behavior for login, scopes,
  stack availability waits, styling, debug output, and hidden product surface.
- `docs/cli-v4/migration-v3-v4.md`: user-facing v3 to v4 command migration.
- `docs/cli-v4/config-format.md`: v4 config shape and credential handling.
- `docs/cli-v4/testing-strategy.md`: expected test coverage and local gates.
- `docs/cli-v4/cutover-plan.md`: future move from `v4/` to the repository
  root.

## Boundaries

- Keep new implementation code under `v4/` until the explicit cutover.
- Do not import root v3 packages from v4 code.
- Keep Cobra command files thin; place runtime and product behavior under
  `v4/internal`.
