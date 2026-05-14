# fctl v3 to v4 Migration

This guide tracks the user-facing migration from the current fctl command tree to
the isolated v4 implementation under `v4/`.

## Core Changes

- Profiles are the primary target selector. A command targets a local,
  self-hosted, Cloud, or Cloud stack profile instead of assuming Formance Cloud.
- API versions are selected by capability resolution. Commands expose product
  intent and use the latest compatible API by default.
- Generated SDK namespaces remain an implementation detail. Command names and
  environment-derived internal identifiers include the service name, for example
  `FCTL_LedgerTransactionList`.
- `auth` remains the canonical service name. Do not introduce `identity` in this
  migration.
- CLI setup and authentication starts at root `login`, with `logout` and
  `whoami` for daily use. `session` remains hidden implementation detail, so
  `auth` is not overloaded between CLI authentication state and the stack Auth
  service.
- `flows` is the canonical command for the former `orchestration` product.
- `cloud stacks` is the canonical command for Cloud stack lifecycle operations.
  `cloud_stacks`, `stack`, and `stacks` are deprecated migration aliases with
  warnings.
- `search` is removed from v4 because the product no longer exists.

## Command Renames

| v3 command | v4 command | Migration behavior |
| --- | --- | --- |
| `--profile <name>` | `--profile <name>` | Primary selector. |
| `--context <name>` | `--profile <name>` | Hidden deprecated alias with a warning. |
| `--organization <org>` / `--stack <stack>` | `--organization <org>` / `--stack <stack>` | Global Cloud/EE target overrides for profiles that support them. |
| `-c`, `--config-dir <dir>` | `-c`, `--config-dir <dir>` | Kept for config directory selection. |
| `-d`, `--debug` | `-d`, `--debug` | Kept; runtime diagnostics are written to stderr. |
| `--insecure-tls` | `--insecure-tls` | Kept as an explicit non-persistent runtime override. |
| none | `--no-color` | New stable flag; v4 renderers are plain by default. |
| `ui` | `cloud ui --print` or `cloud ui` | Root `ui` is a hidden deprecated alias. `--print` is non-browser/non-interactive friendly. |
| `login --membership-uri <url>` | `login --target cloud` or `login --target ee --membership-url <url>` | Browser/device login is deferred; use client credentials or token flags for now. |
| `profiles ...` | `profile ...` | `profiles` is a hidden deprecated alias with a warning. |
| `profiles reset <name>` | `profile unset-defaults <name> --confirm` | Deprecated alias. |
| `profiles set-default-organization <org>` | `profile set --organization <org>` | Deprecated alias updates the current profile. |
| `profiles set-default-stack <stack>` | `profile set --stack <stack>` | Deprecated alias updates the current profile. |
| `orchestration ...` | `flows ...` | `orchestration` is a deprecated alias with a warning. |
| `orchestration workflows create <file>\|-` | `flows workflows create --file <path>\|-` | Deprecated positional file form warns on stderr. |
| `orchestration instances describe` | `flows instances inspect` | `describe` remains a deprecated alias. |
| `payments connectors get-config --connector-id <id>` | `payments connectors config show <connector-id>` | Deprecated alias with a warning. |
| `payments ... get` | `payments ... show` | `get` remains a deprecated alias when cheap to maintain. |
| `reconciliation ... get` | `reconciliation ... show` | `get` remains a deprecated alias. |
| `reconciliation policies reconcile <policy-id> <at-ledger> <at-payments>` | `reconciliation policies reconcile <policy-id> --ledger-at <time> --payments-at <time>` | Deprecated positional timestamp form warns on stderr. |
| `auth clients get` | `auth clients show` | `get` remains a deprecated alias. |
| `auth users get` | `auth users show` | `get` remains a deprecated alias. |
| `auth clients users ...` | `auth users ...` | Deprecated nested alias with a warning. |
| `webhooks change-secret` | `webhooks secret rotate` | `change-secret` remains a deprecated alias. |
| `cloud me info` | `cloud me show` | `info` remains a deprecated alias. |
| `cloud organizations describe` | `cloud organizations show` | `describe` remains a deprecated alias. |
| `cloud organizations authentication-provider configure <type> <name> <client-id> <client-secret>` | `cloud organizations authentication-provider configure --type <type> --name <name> --client-id <client-id> --client-secret-stdin` | Deprecated positional form warns on stderr. |
| `cloud_stacks ...` | `cloud stacks ...` | Deprecated alias with warnings for Cloud lifecycle commands. |
| `stack ...` / `stacks ...` | `cloud stacks ...` | Deprecated aliases with warnings for Cloud lifecycle commands. |
| `search ...` | none | Removed. |

## Argument Changes

| Area | v3 shape | v4 shape |
| --- | --- | --- |
| Wallet credit | `wallets credit <amount> <asset>` | `wallets credit <wallet-id> --amount <amount> --asset <asset>` |
| Wallet debit | `wallets debit <amount> <asset>` | `wallets debit <wallet-id> --amount <amount> --asset <asset>` |
| Payment connector config update | connector type plus `--connector-id` | `payments connectors config update <connector-id> --file <path>\|-` |
| Reconciliation run | positional timestamps | `reconciliation policies reconcile <policy-id> --ledger-at <time> --payments-at <time>` |
| Webhook secret rotation | positional secret | `webhooks secret rotate <config-id> --secret-stdin` or `--secret` |

## Secrets

Secrets should not be passed positionally in new v4 commands. Prefer stdin flags
such as `--secret-stdin` when available. Plain text output must not print clear
secrets returned by APIs. Structured output can expose explicit response fields
when the command is intentionally machine-readable.

## Deferred Items

- Browser/device login in `fctl login` is deferred until the Cloud, EE, and
  generic OIDC device-flow contracts are explicit. Stack commands must remain
  usable without Cloud membership.
- `--telemetry` is deferred until opt-in/out behavior and stored state are
  documented.
- `--quiet` is deferred until each command family defines its primary quiet
  output, so v4 does not expose a silent no-op flag.

## Compatibility Warnings

Deprecated aliases should write warnings to stderr and include the canonical v4
command. They are not long-term compatibility guarantees and can be removed in a
minor v4 release or in v5 depending on usage and maintenance cost.
