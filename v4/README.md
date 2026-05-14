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

## Boundaries

- Keep new implementation code under `v4/` until the explicit cutover.
- Do not import root v3 packages from v4 code.
- Keep Cobra command files thin; place runtime and product behavior under
  `v4/internal`.
