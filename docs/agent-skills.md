# Agent skills

The `formance-fctl-plugin` skill helps an AI coding agent create or extend
embedded fctl service plugins. It covers the public SDK, declarative forms,
API validation, plugin independence and CLI verification.

The versioned sources live in `.agents/skills/formance-fctl-plugin/`. They are the
distribution source; installed copies do not update automatically. Developing
a plugin requires an fctl source checkout and the requested service's API
contract. The skill instructs the agent to inspect those sources before coding.

## Use in this repository

Codex discovers the skill in the repository's `.agents/skills/` directory when
working in this checkout. No personal installation is needed. See the
[official skill discovery documentation](https://learn.chatgpt.com/docs/build-skills#where-codex-loads-local-skills).

For example, with this checkout available:

```text
Use $formance-fctl-plugin to add a service plugin to fctl v4 with declarative
forms and contract tests.
```

## Distribution

Each GoReleaser `.tar.gz` or `.zip` release archive includes:

- `.agents/skills/formance-fctl-plugin/SKILL.md`
- `.agents/skills/formance-fctl-plugin/agents/openai.yaml`
- `.agents/skills/formance-fctl-plugin/references/inputs.md`
- `.agents/skills/formance-fctl-plugin/references/verification.md`
- `docs/agent-skills.md` and `LICENSE`

The directory layout is identical in the repository and an extracted release
archive. The skill is distributed under the repository's MIT license. Extracting
an archive or installing fctl does not install the skill in an agent directory.

## Optional personal installation

From the repository root or an extracted release archive, run this POSIX shell
command to make the skill available in other repositories through the shared
local agent skills directory:

```sh
(
  skill_destination="$HOME/.agents/skills/formance-fctl-plugin"
  if [ -e "$skill_destination" ] || [ -L "$skill_destination" ]; then
    printf 'Skill already exists at %s; compare it before replacing it.\n' "$skill_destination" >&2
    exit 1
  fi
  mkdir -p "$HOME/.agents/skills" &&
    cp -R .agents/skills/formance-fctl-plugin "$skill_destination"
)
```

The command refuses to overwrite an existing directory or symlink. To compare
an installed copy with the distributed source before updating it:

```sh
diff -ru .agents/skills/formance-fctl-plugin "$HOME/.agents/skills/formance-fctl-plugin"
```

Use the installed copy in an agent that supports `SKILL.md` and loads
`~/.agents/skills`. Codex can display both the repository and personal copies
when both are present.

This directory contains development instructions for an AI agent. The service
plugin it helps produce is Go code integrated with fctl through the public SDK.
