# fctl v4 Guidance

Before working on the next major CLI architecture, read:

- `docs/rfcs/0001-fctl-v4-architecture.md`
- `docs/cli-v4/command-design.md`
- `docs/cli-v4/compatibility-manifest.md`
- `todos/01-v4-isolated-skeleton.md`

Core rules:

- Do not couple stack commands to Formance Cloud membership.
- Treat context, target, auth, capabilities, API version, and rendering as separate concepts.
- Commands express product intent; API version selection belongs in the runtime.
- Use `/versions` plus the generated compatibility manifest to select the best supported SDK namespace.
- Keep Cobra as a thin parser/router; keep business logic in typed internal packages.
- Store credentials in a keyring when possible; keep config files free of long-lived secrets.
- Build the rewrite under `v4/` until the explicit cutover goal.
- Follow `todos/*.md` in order unless the user explicitly reprioritizes.
- Commit after each logical step when implementing v4 work.
