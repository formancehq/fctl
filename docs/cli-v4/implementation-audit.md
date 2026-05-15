# fctl v4 Implementation Audit

This audit records the implementation status of `plan.md` so reviewers can
separate completed migration work from explicit blockers and future cutover
work.

## Implemented

- v4 remains isolated under `v4/`; v3 root command behavior has not been moved.
- Profiles/contexts, credentials, auth strategies, runtime target resolution, `/versions`
  parsing, and API version selection are implemented in v4 packages.
- Global flags from the migration plan are implemented when actionable:
  `--profile`, hidden deprecated `--context`, `--organization`, `--stack`,
  `--config-dir/-c`, `--debug/-d`, `--insecure-tls`, `--non-interactive`,
  `--no-color`, and `--output`.
- Root `login`, `logout`, and `whoami` provide the primary user-facing
  authentication flow. `profile` is the primary target-management command, while
  `context` and `session` are hidden from help.
- Interactive `login` uses Charmbracelet Huh when stdin/stderr are terminals.
  Non-TTY runs, `--non-interactive`, and tests keep deterministic plain prompts
  and flag errors.
- Browser/device login is implemented for Cloud and EE. Login does not expose a
  static-token choice in the wizard. Device credentials are stored outside the
  config file and can be refreshed when command scopes are missing.
- `--telemetry` and `--quiet` are intentionally not exposed as silent no-ops.
- Cloud control-plane commands are grouped under `cloud`, with `cloud stacks`
  as the canonical stack lifecycle command and deprecated `cloud_stacks`,
  `stack`, and `stacks` aliases.
- `cloud stacks create` provides interactive name, region, and version prompts,
  sorts stack versions descending, waits for availability by default, and shows
  styled progress and final output.
- Stack data-plane commands do not require Formance Cloud membership.
- Ledger commands use canonical CLI flags and runtime API resolution, including
  v1/v2 adaptation and deprecated flag aliases.
- Payments, wallets, flows, reconciliation, Auth service, and webhooks have v4
  command families with migration aliases for low-cost v3 shapes.
- Human output is styled through shared terminal helpers. JSON/YAML output stays
  stable for scripts. `--debug/-d` emits HTTP diagnostics to stderr with
  sensitive headers redacted.
- The removed `search` product is not exposed in v4.
- `v4/testdata/v3-command-inventory.json` records the v3 Cobra inventory used
  during migration review.
- The user-facing docs under `docs/cli-v4/` describe command reference,
  runtime behavior, migration behavior, compatibility aliases, testing strategy,
  config migration, versioning ownership, and cutover constraints.

## Explicitly Deferred Or Blocked

- `auth login/status/token/logout` are not kept as aliases. `auth` is reserved
  for stack Auth service resources.
- `cloud personal-tokens create` is not implemented because the v3 flow depends
  on Cloud claims, stack access checks, and an Auth token exchange model not yet
  present in the v4 runtime.
- `cloud apps ...` is hidden from visible help because Cloud apps are not part
  of the v4 product surface yet.
- `ledger transactions explain` is hidden until the public stack spec and
  `formance-sdk-go` expose `explainTransaction`.
- Runtime capabilities are still coarse-grained at product/API-namespace level.
  Feature-level manifest checks and intra-namespace capability ranges are
  documented as follow-up work in `docs/cli-v4/versioning-and-ownership.md`.
- Plugins are not the v4 MVP architecture. They remain a future option once
  packaging, discovery, installation, and per-product ownership are explicit.
- A reusable OpenAPI-backed mock server remains future work. Current v4 tests
  use targeted `httptest` handlers that assert method, path, query, body,
  stdout, stderr, errors, and deprecation warnings.
- Cutover from `v4/` to the repository root remains blocked by
  `docs/cli-v4/cutover-plan.md` and must be done in a dedicated goal.

## Verification

Before the latest review commits, the following gates were run repeatedly:

```bash
cd v4 && go test ./...
git diff --check
nix develop --impure --command just pre-commit
```

The v4 tree currently has command integration tests in `v4/cmd/root_test.go`
and focused package tests across config, credentials, auth, capabilities,
runtime, Ledger, Payments, Wallets, Flows, and target proxy behavior. The latest
manual command audit covered 193 visible executable leaf commands with zero
untested visible commands.
