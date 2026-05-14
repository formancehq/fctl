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
- `--telemetry` and `--quiet` are intentionally not exposed as silent no-ops.
- Cloud control-plane commands are grouped under `cloud`, with `cloud stacks`
  as the canonical stack lifecycle command and deprecated `cloud_stacks`,
  `stack`, and `stacks` aliases.
- Stack data-plane commands do not require Formance Cloud membership.
- Ledger commands use canonical CLI flags and runtime API resolution, including
  v1/v2 adaptation, deprecated flag aliases, and v3 preflight behavior for
  `ledger transactions explain`.
- Payments, wallets, flows, reconciliation, Auth service, and webhooks have v4
  command families with migration aliases for low-cost v3 shapes.
- The removed `search` product is not exposed in v4.
- `v4/testdata/v3-command-inventory.json` records the v3 Cobra inventory used
  during migration review.
- The user-facing docs under `docs/cli-v4/` describe command reference,
  migration behavior, compatibility aliases, testing strategy, config migration,
  and cutover constraints.

## Explicitly Deferred Or Blocked

- `auth login/status/token/logout` are not kept as aliases. `auth` is reserved
  for stack Auth service resources.
- `cloud personal-tokens create` is not implemented because the v3 flow depends
  on Cloud claims, stack access checks, and an Auth token exchange model not yet
  present in the v4 runtime.
- `cloud apps versions archive` as a mutating action is not implemented because
  the current deployserver SDK exposes archive read behavior, not the mutating
  archive operation.
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
```

The v4 tree currently has command integration tests in `v4/cmd/root_test.go`
and focused package tests across config, credentials, auth, capabilities,
runtime, Ledger, Payments, Wallets, Flows, and target proxy behavior.
