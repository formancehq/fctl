# fctl v4 Compatibility Aliases

Aliases in v4 are migration aids. They should be cheap to maintain, visible in
tests, and explicit about their canonical replacement.

| Alias | Canonical command | Status | Removal target |
| --- | --- | --- | --- |
| `orchestration ...` | `flows ...` | Deprecated warning | v4.x or v5 |
| `flows instances describe` | `flows instances inspect` | Deprecated warning | v4.x |
| `payments connectors config get` | `payments connectors config show` | Deprecated warning | v4.x |
| `payments connectors update-config` | `payments connectors config update` | Deprecated warning | v4.x |
| `payments transfer-initiation get` | `payments transfer-initiation show` | Deprecated warning | v4.x |
| `payments transfer-initiation update_status` | `payments transfer-initiation update-status` | Deprecated warning | v4.x |
| `reconciliation get` | `reconciliation show` | Deprecated warning | v4.x |
| `reconciliation policies get` | `reconciliation policies show` | Deprecated warning | v4.x |
| `auth clients get` | `auth clients show` | Deprecated warning | v4.x |
| `auth users get` | `auth users show` | Deprecated warning | v4.x |
| `webhooks change-secret` | `webhooks secret rotate` | Deprecated warning | v4.x |
| `stack ...` | `cloud_stacks ...` | Planned deprecated warning | v4.x or v5 |
| `stacks ...` | `cloud_stacks ...` | Planned deprecated warning | v4.x or v5 |

## Removed Commands

| Command | Status | Reason |
| --- | --- | --- |
| `search ...` | Removed | The product no longer exists. |
| `se` | Removed | Alias for removed `search`. |

## Rules

- Alias warnings go to stderr.
- Warning text must name the canonical command.
- Aliases should not hide important v4 target arguments such as wallet IDs or
  connector IDs.
- Service-qualified identifiers remain service-qualified. Do not add generic
  internal aliases such as `FCTL_TransactionList`.
