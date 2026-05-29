# Goal 10 - integration tests and UX hardening

```text
/goal
Harden v4 CLI behavior with integration-style tests and user-facing error/output polish.

Read first:
- docs/rfcs/0001-fctl-v4-architecture.md
- docs/cli-v4/command-design.md
- all prior todos that have been completed.

Deliverables:
- testscript-style integration tests or equivalent command execution harness.
- stable stdout/stderr assertions for context, inspect, auth, and first Ledger command.
- consistent error types and exit codes for unsupported target, missing auth, unsupported API, and invalid config.
- JSON/YAML output checks.
- non-interactive mode checks.

Constraints:
- do not require real Formance Cloud.
- prefer fake local HTTP servers or fixtures.
- do not add broad refactors unrelated to UX/test hardening.
- commit test harness first, then behavior changes in small commits.
- run git diff --check before each commit.

Tests:
- go test ./... in v4.
- integration harness runs in CI-friendly mode.

Done when:
- core v4 workflows are covered as real CLI usage, not only unit tests.
```
