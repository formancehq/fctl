# fctl v4 Documentation

The v4 CLI is implemented under `v4/`, but its documentation lives here so it
can be reviewed with the repository-level architecture docs.

Read these in order when onboarding to v4:

1. `../rfcs/0001-fctl-v4-architecture.md` for the target architecture.
2. `../adr/` for accepted decisions.
3. `command-design.md` for command shape and Cobra boundaries.
4. `config-format.md` for profiles, contexts, and credential storage.
5. `runtime-behavior.md` for current login, scopes, stack waits, rendering, and
   debug behavior.
6. `command-reference.md` for visible command families.
7. `migration-v3-v4.md` and `compatibility-aliases.md` for v3 compatibility.
8. `testing-strategy.md` for local and CI validation.
9. `implementation-audit.md` for current implementation status and known gaps.
10. `cutover-plan.md` for the future move out of `v4/`.

Keep `command-reference.md` and `runtime-behavior.md` aligned with Cobra help and
the manual command audit whenever commands are added, hidden, or renamed.
