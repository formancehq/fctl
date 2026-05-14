# Goal 09 - v3 config migration

```text
/goal
Implement explicit v3 to v4 configuration migration.

Read first:
- docs/cli-v4/migration-from-v3.md
- docs/cli-v4/config-format.md
- docs/adr/0001-contexts-as-primary-target.md

Deliverables:
- fctl v4 config migrate-v3 command.
- read-only parser for existing v3 config/profile files.
- migration planner that shows contexts and credential moves before writing.
- migration writer that creates v4 config and stores credentials through the credentials interface when possible.
- dry-run mode.
- tests with fixture v3 configs.

Constraints:
- do not delete or mutate v3 files.
- do not silently migrate during normal command execution.
- never print secrets in logs or normal output.
- commit parser, planner, writer, and command in separate reviewable commits.
- run git diff --check before each commit.

Tests:
- unit tests for v3 fixture parsing.
- unit tests for migration plan output.
- command-level tests for dry-run and write mode.
- go test ./... in v4.

Done when:
- users can explicitly migrate v3 profiles to v4 contexts without losing v3 data.
```
