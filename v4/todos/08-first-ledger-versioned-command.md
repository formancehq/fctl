# Goal 08 - first Ledger versioned command

```text
/goal
Implement the first Ledger command using versioned handlers.

Read first:
- docs/cli-v4/command-design.md
- docs/cli-v4/compatibility-manifest.md
- docs/adr/0003-api-version-resolution.md

Suggested command:
- fctl v4 ledger transactions list

Deliverables:
- canonical input model for listing Ledger transactions.
- versioned handlers for the SDK namespaces available in the public SDK.
- adapters from canonical input to generated SDK request types.
- runtime selection of the best compatible handler.
- JSON and table rendering.
- clear error when no compatible API version exists.

Constraints:
- do not migrate all Ledger commands.
- avoid exposing v1/v2 in the primary command path.
- support --api-version for pinning if the runtime already supports it.
- keep Cobra thin.
- commit adapters, runtime wiring, and command tests separately where practical.
- run git diff --check before each commit.

Tests:
- unit tests for canonical input parsing.
- unit tests for handler selection.
- command-level tests using fake SDK/runtime boundaries.
- go test ./... in v4.

Done when:
- one Ledger command proves the versioned command pattern end to end.
```
