# Goal 03 - v4 context commands

```text
/goal
Add the first v4 context management commands.

Read first:
- docs/cli-v4/config-format.md
- docs/adr/0001-contexts-as-primary-target.md
- v4/internal/config and v4/internal/runtime from prior goals.

Deliverables:
- fctl v4 context list.
- fctl v4 context show [name].
- fctl v4 context use <name>.
- fctl v4 context create stack <name> --stack-url ... with minimal auth reference support.
- JSON output support for these commands.
- tests for config mutation and command output.

Constraints:
- commands must be non-destructive.
- no Cloud membership dependency.
- support non-interactive usage.
- keep Cobra command files thin.
- commit after each command group or test group.
- run git diff --check before each commit.

Tests:
- go test ./... in v4.
- command-level tests for list/show/use/create.

Done when:
- contexts can be created, listed, inspected, and selected in v4 config.
- command output is stable enough for scripts.
```
