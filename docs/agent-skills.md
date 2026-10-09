# Agent skills

The `formance-fctl-plugin` skill helps an AI coding agent create or extend
embedded fctl service plugins. It covers the public SDK, declarative forms,
API validation, plugin independence and CLI verification.

The versioned sources live in `skills/formance-fctl-plugin/`. They are the
distribution source; installed copies do not update automatically. Developing
a plugin requires an fctl source checkout and the requested service's API
contract. The skill instructs the agent to inspect those sources before coding.

## Distribution

Each GoReleaser `.tar.gz` or `.zip` release archive includes:

- `skills/formance-fctl-plugin/SKILL.md`
- `skills/formance-fctl-plugin/agents/openai.yaml`
- `skills/formance-fctl-plugin/references/inputs.md`
- `skills/formance-fctl-plugin/references/verification.md`
- `docs/agent-skills.md` and `LICENSE`

The directory layout is identical in the repository and an extracted release
archive. The skill is distributed under the repository's MIT license. Extracting
an archive or installing fctl does not install the skill in an agent directory.

## Installation

From the repository root or an extracted release archive, run this POSIX shell
command to install the skill in the shared local agent skills directory:

```sh
(
  skill_destination="$HOME/.agents/skills/formance-fctl-plugin"
  if [ -e "$skill_destination" ] || [ -L "$skill_destination" ]; then
    printf 'Skill already exists at %s; compare it before replacing it.\n' "$skill_destination" >&2
    exit 1
  fi
  mkdir -p "$HOME/.agents/skills" &&
    cp -R skills/formance-fctl-plugin "$skill_destination"
)
```

The command refuses to overwrite an existing directory or symlink. To compare
an installed copy with the distributed source before updating it:

```sh
diff -ru skills/formance-fctl-plugin "$HOME/.agents/skills/formance-fctl-plugin"
```

Use the skill in an agent that supports `SKILL.md` and loads
`~/.agents/skills`. For example, with an fctl checkout available:

```text
Use $formance-fctl-plugin to add a service plugin to fctl v4 with declarative
forms and contract tests.
```

This directory contains development instructions for an AI agent. The service
plugin it helps produce is Go code integrated with fctl through the public SDK.
