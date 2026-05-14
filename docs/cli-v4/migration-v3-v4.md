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
- `cloud_stacks` is the canonical command for Cloud stack lifecycle operations.
  `stack` and `stacks` are deprecated migration aliases with warnings.
- `search` is removed from v4 because the product no longer exists.

## Command Renames

| v3 command | v4 command | Migration behavior |
| --- | --- | --- |
| `orchestration ...` | `flows ...` | `orchestration` is a deprecated alias with a warning. |
| `orchestration instances describe` | `flows instances inspect` | `describe` remains a deprecated alias. |
| `payments ... get` | `payments ... show` | `get` remains a deprecated alias when cheap to maintain. |
| `reconciliation ... get` | `reconciliation ... show` | `get` remains a deprecated alias. |
| `auth clients get` | `auth clients show` | `get` remains a deprecated alias. |
| `auth users get` | `auth users show` | `get` remains a deprecated alias. |
| `webhooks change-secret` | `webhooks secret rotate` | `change-secret` remains a deprecated alias. |
| `cloud me info` | `cloud me show` | `info` remains a deprecated alias. |
| `cloud organizations describe` | `cloud organizations show` | `describe` remains a deprecated alias. |
| `stack ...` / `stacks ...` | `cloud_stacks ...` | Deprecated aliases with warnings for Cloud lifecycle commands. |
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

## Compatibility Warnings

Deprecated aliases should write warnings to stderr and include the canonical v4
command. They are not long-term compatibility guarantees and can be removed in a
minor v4 release or in v5 depending on usage and maintenance cost.
