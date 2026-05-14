# Goal 02 - v4 foundation packages

```text
/goal
Implement the v4 foundation packages without migrating product commands.

Read first:
- docs/rfcs/0001-fctl-v4-architecture.md
- docs/cli-v4/config-format.md
- docs/cli-v4/compatibility-manifest.md
- docs/adr/0001-contexts-as-primary-target.md
- docs/adr/0002-auth-is-decoupled-from-cloud.md
- docs/adr/0003-api-version-resolution.md

Deliverables:
- v4/internal/config for versioned context config structs, load, save, validate, current context resolution, and env/flag override hooks.
- v4/internal/credentials with interfaces for secret get/set/delete and an explicit insecure file implementation for development tests.
- v4/internal/capabilities with APIVersion, Product, Feature, Manifest, ComponentCompatibility, and resolver data types.
- v4/internal/runtime with a typed Runtime shell that resolves config, context, target, and API policy.
- unit tests for config validation, context selection, and compatibility range resolution.

Constraints:
- do not add real auth flows yet.
- do not migrate existing v3 commands.
- do not store long-lived secrets in config structs except secret references.
- commit after each package or cohesive test set.
- run git diff --check before each commit.

Tests:
- go test ./... in v4.
- targeted unit tests for each new package.

Done when:
- the foundation packages compile and are tested.
- no v3 behavior changes.
- commits are small and reviewable.
```
