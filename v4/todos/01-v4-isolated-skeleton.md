# Goal 01 - v4 isolated skeleton

```text
/goal
Create the isolated fctl v4 skeleton under the top-level v4/ directory.

Read first:
- AGENTS.md
- docs/rfcs/0001-fctl-v4-architecture.md
- docs/adr/0005-build-v4-in-isolated-directory.md
- docs/cli-v4/config-format.md

Deliverables:
- v4 Go module or buildable v4 application skeleton.
- v4/cmd root command with Cobra wired as a thin parser/router.
- v4/internal package directories for config, credentials, capabilities, runtime, commands, render, and prompt.
- minimal v4 main entrypoint that can print version/help.
- documentation note explaining how to build/run v4 without touching v3.

Constraints:
- stay on branch feat/v4.
- do not modify or delete existing v3 command behavior.
- do not move root files yet.
- keep new code isolated under v4/.
- commit each logical step separately.
- run git diff --check before each commit.

Tests:
- run go test for the v4 module/package if a module is created.
- run the v4 binary help/version command if buildable.

Done when:
- v4 can be built or at least tested independently.
- v3 root remains untouched except documentation/build notes if needed.
- all changes are committed in small reviewable commits.
```
