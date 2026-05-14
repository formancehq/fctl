# Goal 05 - capabilities manifest generator

```text
/goal
Generate the v4 compatibility manifest from the stack OpenAPI spec.

Read first:
- docs/cli-v4/compatibility-manifest.md
- docs/adr/0003-api-version-resolution.md

Reference spec:
- https://github.com/formancehq/stack/releases/download/v3.2.4/generate.json

Deliverables:
- generator script or Go tool that reads generate.json.
- generated v4/internal/capabilities manifest containing spec version, products, API namespaces, operation IDs, HTTP methods, paths, and tags.
- manual component version compatibility table kept separate from generated data.
- test fixture based on a reduced OpenAPI sample.
- documentation for regenerating the manifest.

Constraints:
- generated files must be deterministic.
- do not require network in normal tests.
- keep manual compatibility ranges small and explicit.
- commit generator and generated output separately if useful for review.
- run git diff --check before each commit.

Tests:
- unit tests for generator parsing.
- go test ./... in v4.
- optional check that generated output is up to date.

Done when:
- v4 has generated operation metadata from OpenAPI and manual component compatibility ranges.
```
