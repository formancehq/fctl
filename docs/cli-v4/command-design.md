# fctl v4 Command Design

Commands should express Formance product intent, not OpenAPI or SDK structure.

Prefer:

```bash
fctl ledger transactions list
fctl ledger transactions revert <id>
fctl ledger schemas insert <version>
```

Avoid:

```bash
fctl ledger v2 transactions list
fctl ledger transactions list-v2
```

## Canonical Inputs

Each command should parse into a version-independent input model.

```go
type ListTransactionsInput struct {
    Ledger         string
    AccountAddress string
    PageSize       int64
}
```

Version-specific adapters convert that model into generated SDK request types.

```go
func toLedgerV1(input ListTransactionsInput) operations.ListTransactionsRequest
func toLedgerV2(input ListTransactionsInput) operations.V2ListTransactionsRequest
```

## Renamed API Parameters

If API v1 calls a field `account` and API v2 calls it `address`, but the CLI concept is the same, expose one canonical flag.

```bash
fctl ledger transactions list --account users:123
```

Keep aliases only for CLI compatibility, not because generated API names changed.

## Version-Specific Features

A command can exist even if only newer targets support it.

```bash
fctl ledger transactions explain <id>
```

If the current target only supports an older Ledger API, return:

```text
ledger transactions explain requires Ledger >= 3.0.0.
Current target runs Ledger 2.3.4.
```

## Help Text

Help should be stable and product-oriented. For flags or commands requiring newer APIs, include capability notes:

```text
--include-descendants    Include child accounts (requires ledger API v2+)
```

Context-aware help can be added later, but the base help should remain useful without network calls.
