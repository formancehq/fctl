# fctl v4 Compatibility Manifest

The v4 CLI should derive most operation metadata from the released stack OpenAPI document.

Current reference:

```text
https://github.com/formancehq/stack/releases/download/v3.2.4/generate.json
```

The spec contains versioned tags such as:

- `ledger.v1`
- `ledger.v2`
- `payments.v1`
- `payments.v3`
- `orchestration.v1`
- `orchestration.v2`
- `auth.v1`
- `wallets.v1`
- `webhooks.v1`
- `reconciliation.v1`

Retired products such as `search` must be ignored even if an old stack spec still
contains their tags.

## Generated Data

Generate a manifest with:

- stack spec version
- product name
- API namespace
- operation ID
- HTTP method
- path
- tags

Example shape:

```json
{
  "specVersion": "v3.2.4",
  "products": {
    "ledger": {
      "apiVersions": ["v1", "v2"],
      "operations": {
        "listTransactions": {
          "v1": {
            "operationId": "listTransactions",
            "path": "/api/ledger/{ledger}/transactions"
          },
          "v2": {
            "operationId": "v2ListTransactions",
            "path": "/api/ledger/v2/{ledger}/transactions"
          }
        }
      }
    }
  }
}
```

## Manual Data

The OpenAPI spec does not fully define which component binary versions support which API namespaces. Keep that as a small explicit table.

Example:

```go
var ComponentCompatibility = []ComponentRange{
    {
        Product: "ledger",
        Range: ">=1.0.0 <2.0.0",
        APIVersions: []APIVersion{"v1"},
    },
    {
        Product: "ledger",
        Range: ">=2.0.0 <3.0.0",
        APIVersions: []APIVersion{"v1", "v2"},
    },
}
```

## Runtime Resolution

1. Call `GetVersions`.
2. Convert component versions into supported API namespaces.
3. Find command handlers for the requested feature.
4. Select the highest compatible API namespace unless the user pinned one.
5. Return a clean error if no handler can run.
