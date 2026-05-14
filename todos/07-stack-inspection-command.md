# Goal 07 - first stack inspection command

```text
/goal
Add a first non-destructive v4 stack inspection command to validate runtime resolution.

Read first:
- docs/rfcs/0001-fctl-v4-architecture.md
- docs/cli-v4/compatibility-manifest.md
- v4/internal/runtime from prior goals.

Deliverables:
- fctl v4 target inspect or fctl v4 capabilities inspect.
- command calls /versions for the current stack target.
- output includes target URL, component versions, health, inferred API namespaces, and API policy.
- JSON output support.
- command-level tests with fake versions response.

Constraints:
- no Cloud membership requirement.
- no mutation of remote state.
- command must work against local/self-hosted contexts.
- commit command, renderers, and tests in reviewable chunks.
- run git diff --check before each commit.

Tests:
- go test ./... in v4.
- command-level tests for table and JSON output.

Done when:
- v4 can inspect a configured stack and show inferred capabilities without touching Ledger data.
```
