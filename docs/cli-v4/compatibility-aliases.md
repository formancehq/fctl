# fctl v4 Compatibility Aliases

Aliases in v4 are migration aids. They should be cheap to maintain, visible in
tests, and explicit about their canonical replacement.

| Alias | Canonical command | Status | Removal target |
| --- | --- | --- | --- |
| `--profile <name>` | `--context <name>` | Deprecated warning | v4.x |
| `profiles ...` | `context ...` | Deprecated warning | v4.x |
| `orchestration ...` | `flows ...` | Deprecated warning | v4.x or v5 |
| `orchestration workflows create <file>\|-` | `flows workflows create --file <path>\|-` | Deprecated warning | v4.x |
| `flows instances describe` | `flows instances inspect` | Deprecated warning | v4.x |
| `payments connectors get-config --connector-id <id>` | `payments connectors config show <connector-id>` | Deprecated warning | v4.x |
| `payments connectors config get` | `payments connectors config show` | Deprecated warning | v4.x |
| `payments connectors update-config` | `payments connectors config update` | Deprecated warning | v4.x |
| `payments transfer-initiation get` | `payments transfer-initiation show` | Deprecated warning | v4.x |
| `payments transfer-initiation update_status` | `payments transfer-initiation update-status` | Deprecated warning | v4.x |
| `reconciliation get` | `reconciliation show` | Deprecated warning | v4.x |
| `reconciliation policies get` | `reconciliation policies show` | Deprecated warning | v4.x |
| `reconciliation policies reconcile <policy-id> <at-ledger> <at-payments>` | `reconciliation policies reconcile <policy-id> --ledger-at --payments-at` | Deprecated warning | v4.x |
| `auth clients get` | `auth clients show` | Deprecated warning | v4.x |
| `auth users get` | `auth users show` | Deprecated warning | v4.x |
| `auth clients users ...` | `auth users ...` | Deprecated warning | v4.x |
| `webhooks change-secret` | `webhooks secret rotate` | Deprecated warning | v4.x |
| `cloud me info` | `cloud me show` | Deprecated warning | v4.x |
| `cloud organizations describe` | `cloud organizations show` | Deprecated warning | v4.x |
| `cloud organizations authentication-provider configure <type> <name> <client-id> <client-secret>` | `cloud organizations authentication-provider configure --type --name --client-id --client-secret-stdin` | Deprecated warning | v4.x |
| `cloud_stacks ...` | `cloud stacks ...` | Deprecated warning | v4.1, v4.2, or v5 |
| `stack ...` | `cloud stacks ...` | Deprecated warning | v4.1, v4.2, or v5 |
| `stacks ...` | `cloud stacks ...` | Deprecated warning | v4.1, v4.2, or v5 |

## Removed Commands

| Command | Status | Reason |
| --- | --- | --- |
| `login --membership-uri <url>` | Removed | CLI authentication moved to `session login ...`, and no alias is kept to avoid mixing CLI session state with the Auth service. |
| `auth login/status/token/logout` | Removed | `auth` is reserved for stack Auth service resources; CLI session commands live under `session`. |
| `search ...` | Removed | The product no longer exists. |
| `se` | Removed | Alias for removed `search`. |

## Rules

- Alias warnings go to stderr.
- Warning text must name the canonical command.
- Aliases should not hide important v4 target arguments such as wallet IDs or
  connector IDs.
- Service-qualified identifiers remain service-qualified. Do not add generic
  internal aliases such as `FCTL_TransactionList`.
