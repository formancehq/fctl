# Goal 11 - v4 cutover plan

```text
/goal
Prepare the final cutover plan for moving the completed v4 implementation from v4/ to the repository root.

Read first:
- docs/adr/0005-build-v4-in-isolated-directory.md
- docs/rfcs/0001-fctl-v4-architecture.md
- all completed todos.

Deliverables:
- written cutover checklist.
- inventory of root files to delete, move, or preserve.
- module path and import path plan.
- release and packaging plan.
- compatibility notes for users and contributors.
- final validation matrix before the cutover commit.

Constraints:
- this goal is planning only unless the user explicitly asks to execute cutover.
- do not delete v3 root files during this goal.
- do not move v4 to root yet.
- commit the cutover plan as documentation.
- run git diff --check before commit.

Tests:
- documentation-only unless build metadata is touched.

Done when:
- the team has an explicit, reviewable checklist for the final root replacement.
```
