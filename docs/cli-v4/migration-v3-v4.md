# fctl v3 to v4 Migration

This guide tracks the user-facing migration from the current fctl command tree to
the isolated v4 implementation under `v4/`.

## Core Changes

- Contexts are the primary target selector. A command targets a local,
  self-hosted, Cloud, or Cloud stack context instead of assuming Formance Cloud.
- API versions are selected by capability resolution. Commands expose product
  intent and use the latest compatible API by default.
- Generated SDK namespaces remain an implementation detail. Command names and
  environment-derived internal identifiers include the service name, for example
  `FCTL_LedgerTransactionList`.
- `auth` remains the canonical service name. Do not introduce `identity` in this
  migration.
- `flows` is the canonical command for the former `orchestration` product.
- `cloud stacks` is the canonical command for Cloud stack lifecycle operations.
  `cloud_stacks`, `stack`, and `stacks` are deprecated migration aliases with
  warnings.
- `search` is removed from v4 because the product no longer exists.

## Command Renames

| v3 command | v4 command | Migration behavior |
| --- | --- | --- |
| `--profile <name>` | `--context <name>` | `--profile` is a deprecated alias with a warning. |
| `-c`, `--config-dir <dir>` | `-c`, `--config-dir <dir>` | Kept for config directory selection; `--context` has no short flag. |
| `-d`, `--debug` | `-d`, `--debug` | Kept; runtime diagnostics are written to stderr. |
| `--insecure-tls` | `--insecure-tls` | Kept as an explicit non-persistent runtime override. |
| none | `--no-color` | New stable flag; v4 renderers are plain by default. |
| `ui` | `ui --print` or `ui` | Kept for Cloud contexts; `--print` is non-browser/non-interactive friendly. |
| `profiles ...` | `context ...` | `profiles` is a deprecated alias with a warning. |
| `profiles reset <name>` | `context unset-defaults <name> --confirm` | `profiles reset` is a deprecated alias with a warning. |
| `profiles set-default-organization <org>` | `context set --organization <org>` | Deprecated alias updates the current context. |
| `profiles set-default-stack <stack>` | `context set --stack <stack>` | Deprecated alias updates the current context. |
| `orchestration ...` | `flows ...` | `orchestration` is a deprecated alias with a warning. |
| `orchestration instances describe` | `flows instances inspect` | `describe` remains a deprecated alias. |
| `payments ... get` | `payments ... show` | `get` remains a deprecated alias when cheap to maintain. |
| `reconciliation ... get` | `reconciliation ... show` | `get` remains a deprecated alias. |
| `auth clients get` | `auth clients show` | `get` remains a deprecated alias. |
| `auth users get` | `auth users show` | `get` remains a deprecated alias. |
| `webhooks change-secret` | `webhooks secret rotate` | `change-secret` remains a deprecated alias. |
| `cloud me info` | `cloud me show` | `info` remains a deprecated alias. |
| `cloud organizations describe` | `cloud organizations show` | `describe` remains a deprecated alias. |
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

- `auth login cloud` is deferred until the Cloud device/browser login contract is
  explicit in v4. Stack commands must remain usable without Cloud membership.
- `auth login oidc` is deferred until the generic device-flow contract is
  specified. Use `auth login client-credentials` for machine-to-machine auth.
- `--telemetry` is deferred until opt-in/out behavior and stored state are
  documented.

## Compatibility Warnings

Deprecated aliases should write warnings to stderr and include the canonical v4
command. They are not long-term compatibility guarantees and can be removed in a
minor v4 release or in v5 depending on usage and maintenance cost.
