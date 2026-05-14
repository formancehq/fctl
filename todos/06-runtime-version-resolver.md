# Goal 06 - runtime API version resolver

```text
/goal
Implement runtime API version resolution using /versions and the compatibility manifest.

Read first:
- docs/cli-v4/compatibility-manifest.md
- docs/adr/0003-api-version-resolution.md
- v4/internal/capabilities from prior goals.

Deliverables:
- Runtime support for calling SDK GetVersions or an abstract versions client.
- parser for /versions response into component versions.
- resolver that maps component version -> supported API namespaces.
- resolver that selects the highest compatible command handler unless pinned by policy.
- support for policies: latest-compatible, pinned, latest if feasible.
- clean unsupported-feature errors.

Constraints:
- make resolver testable without network.
- do not add Ledger command migration yet.
- avoid probing endpoints as a substitute for /versions.
- commit resolver model, implementation, and tests in small chunks.
- run git diff --check before each commit.

Tests:
- unit tests for component version mapping.
- unit tests for handler selection.
- unit tests for pinned version errors.
- go test ./... in v4.

Done when:
- a command can ask runtime for the correct API handler based on target versions and policy.
```
