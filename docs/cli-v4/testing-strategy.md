# fctl v4 Testing Strategy

The v4 CLI should remain scriptable, version-aware, and safe for local,
self-hosted, and Cloud targets.

## Unit Tests

| Area | Coverage |
| --- | --- |
| `internal/config` | Config parsing, defaults, context validation, v3 migration idempotence. |
| `internal/credentials` | In-memory store, insecure file fallback, delete/read errors, file modes. |
| `internal/auth` | No-auth, token refs, stdin/env token sources, client credentials token flow. |
| `internal/capabilities` | `/versions` parsing, semver ranges, latest compatible selection, pinned API errors. |
| `internal/runtime` | Context resolution, auth wiring, target versions, API policy. |
| `internal/commands/*` | Canonical inputs, version-specific SDK adapters, validation errors. |
| `internal/render` | Plain/json/yaml behavior and stdout/stderr separation. |

## CLI Integration Tests

Use `httptest` servers per command family. Each test should:

- create a temporary v4 config directory;
- create or select an explicit context;
- expose `/versions` with the product under test;
- assert the exact API path, method, query, and request body;
- assert stdout, stderr, and returned errors;
- verify deprecated aliases warn on stderr;
- verify destructive commands require `--confirm`;
- verify plain output does not print secrets.
- verify human output uses the shared styled renderers for success, info,
  warnings, key/value blocks, and tables.
- verify hidden commands stay out of visible help and visible command audits.

## Mock Stack

The OpenAPI stack spec is available at:

```text
https://github.com/formancehq/stack/releases/download/v3.2.4/generate.json
```

A future mock server can be generated under `v4/internal/testserver/openapi`.
Until then, targeted `httptest` handlers are preferred because they keep each
command migration small and reviewable.

## Gates Before Commit

Run from the repository root unless noted:

```bash
cd v4 && go test ./...
git diff --check
nix develop --impure --command just pre-commit
```

Only stage the v4 files and documentation for the current logical change. Do not
stage unrelated v3 changes or pre-existing todo deletions.

## Manual Command Audit

For broad command-surface changes, build the v4 binary and audit visible leaf
commands from Cobra help/completion. The audit should exclude hidden commands and
the synthetic `help <command>` subtree.

Each visible leaf command should be covered by one of:

- a real command execution against a safe Cloud or stack target;
- a local fake Cloud/stack server for mutating or destructive commands;
- a documented expected error when the selected real target does not expose that
  component, followed by local fake-server coverage for the command behavior.

The audit output should record:

- visible executable leaf command count;
- `--help` routing failures;
- covered OK commands;
- commands unavailable on the selected real target;
- commands intentionally hidden until supported;
- commands not manually executed yet.
