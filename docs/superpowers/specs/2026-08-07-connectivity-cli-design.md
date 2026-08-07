# Connectivity CLI Design

**Date:** 2026-08-07
**Status:** Approved in conversation; pending written-spec review
**Upstream contract:** `formancehq/connectivity@develop` commit `2ec12564c02c040b3f9da5fbdf40c880e67c1137`, Connectivity API `0.1.0`

## Goal

Add a first-class `fctl connectivity` module that exposes the Connectivity
plugin catalog and instance lifecycle as domain-oriented commands. The module
must feel native to `fctl`, use the selected stack and existing authentication,
and provide reliable context-aware shell completion.

Connectivity is a new module and may eventually replace Payments. This change
does not remove, alias, or modify any `fctl payments` command.

## Scope

The upstream API provides health endpoints, read-only plugin catalog endpoints,
and CRUD endpoints for instances. The CLI exposes the user workflows, not the
HTTP verbs:

```text
fctl connectivity plugins list
fctl connectivity plugins show <plugin>

fctl connectivity instances list
fctl connectivity instances show <instance>
fctl connectivity instances install <plugin>
fctl connectivity instances configure <instance>
fctl connectivity instances uninstall <instance>
```

The health probes remain operational endpoints and do not receive CLI commands.
The API's full-replacement `PUT /instances/{name}` is not exposed in this first
version because there is no corresponding `fctl` workflow. A future declarative
`apply` command can add that behavior without changing the commands above.

## Command conventions

The hierarchy follows the dominant `fctl <module> <resource> <action>` grammar.
The root uses `fctl.NewStackCommand`, so `--organization` and `--stack` behave
like they do for Ledger and Payments.

### Plugins

`plugins list` calls `GET /plugins`. It supports the standard `--page-size` and
`--cursor` flags and maps them to the API's `limit` and `continue` parameters.
Plain output contains Name, Default Version, Description, Capabilities, and
Phase. JSON output preserves the complete plugin objects and continuation data
inside the standard `fctl` data envelope.

`plugins show <plugin>` calls `GET /plugins/{name}`. Plain output includes the
plugin metadata, image, version choices, capabilities, status, documentation
URL, and a readable summary of its configuration schema. JSON output preserves
the complete plugin object.

Aliases:

- `plugins`: `plugin`, `p`
- `list`: `ls`, `l`
- `show`: `get`, `g`, `sh`, `s`

### Instances

`instances list` calls `GET /instances`. It supports `--plugin`, `--page-size`,
and `--cursor`. Plain output contains Name, Plugin, Version, Ledger, Phase,
State, Current Sequence, Source Tip Sequence, and Last Error. JSON output
preserves the complete instances and continuation data.

`instances show <instance>` calls `GET /instances/{name}`. Plain output shows
metadata, desired specification, effective image, lifecycle state, ingestion
progress, and error information. It lists configuration keys and their source
types but never prints inline configuration values. JSON output preserves the
API response, consistent with the existing `fctl -o json` contract.

`instances install <plugin>` calls `GET /plugins/{plugin}` to obtain defaults
and the configuration schema, validates the requested configuration, and then
calls `POST /instances`. It accepts:

- `--name`, defaulting to the plugin name;
- required `--ledger`;
- optional `--version`, `--start-sequence`, and `--poll-interval`;
- optional `--config <file>`;
- repeatable `--env-file <file>`;
- repeatable `--set KEY=VALUE`;
- the standard confirmation flag.

`instances configure <instance>` calls `GET /instances/{name}`, then
`GET /plugins/{plugin}` for schema-aware parsing, and finally
`PATCH /instances/{name}` with `application/merge-patch+json`. It accepts the
same configuration inputs as install. It also accepts optional `--version`,
`--ledger`, `--start-sequence`, and `--poll-interval`; only explicitly supplied
fields are patched. At least one change must be supplied.

`instances uninstall <instance>` calls `DELETE /instances/{name}` after the
standard `fctl` confirmation flow.

Aliases:

- `instances`: `instance`, `i`
- `list`: `ls`, `l`
- `show`: `get`, `g`, `sh`, `s`
- `install`: `create`, `in`
- `configure`: `config`, `update`, `c`
- `uninstall`: `delete`, `remove`, `rm`, `u`

## Configuration inputs

The configuration model follows Connectivity's `InstanceConfig` schema and the
UX already proven by the experimental `cctl` on `connectivity@develop`.

`--set` accepts these value sources:

```text
--set KEY=value
--set KEY=@path/to/local/file
--set KEY=@-
--set KEY=secret://secret-name/key
--set KEY=configmap://config-map-name/key
```

For `@path`, the local file contents become the inline value. `@-` reads stdin
through the existing `fctl.ReadFile` behavior and therefore requires
`--confirm`. Secret and ConfigMap forms create references and do not read the
referenced Kubernetes objects.

`--env-file` accepts dotenv-style `KEY=VALUE` records, ignores blank lines and
comments, and is repeatable. It represents environment entries only.

`--config` accepts YAML or JSON. A flat mapping is interpreted as environment
values. A structured document may contain `env` and `files` matching the
Connectivity API's `InstanceConfig` shape.

During install, values merge in this exact order, from lowest to highest
precedence:

1. plugin defaults;
2. `--config`;
3. `--env-file` values, with later files winning;
4. `--set` values, with later occurrences winning.

During configure, the instance's current configuration replaces plugin
defaults as the lowest-precedence base. This preserves every untouched env
entry and file mount. The command sends the resulting complete `config` object
inside the merge patch, so changing one file mount cannot accidentally replace
the other mounts in the API's array-valued `files` field.

The plugin's `configSchema` determines whether a key is an environment entry or
a file mount. Required keys are validated before the write request. Invalid
assignments, malformed references, unreadable files, unknown schema keys, and
missing required values return actionable errors. Password-formatted values are
never included in plain terminal output.

## Client architecture

A focused internal Connectivity client owns the API models and the seven
non-health operations used by the commands. It is handwritten rather than
generated because the contract is small, the upstream API is still `0.1.0`, and
generated code would add substantial repository and toolchain surface.

The client accepts a base URL and `http.Client`, exposes an interface used by
the commands, escapes path and query values, and decodes the API's structured
`{code,message,details}` errors. Non-success responses include the HTTP status
and structured API error when available.

Production wiring uses `fctl.NewStackClientsFromFlags`. Requests target:

```text
<selected-stack-uri>/api/connectivity/plugins
<selected-stack-uri>/api/connectivity/instances
```

This reuses the existing OAuth token source, refresh behavior, transport,
debugging, TLS settings, and `fctl/<version>` user agent. No Connectivity-specific
profile, context, token, or API URL flag is introduced.

## Package boundaries

- `internal/connectivityclient` contains HTTP models, the client interface,
  implementation, and structured errors.
- `cmd/connectivity` wires the stack-scoped root and shared client factory.
- `cmd/connectivity/plugins` owns plugin commands, stores, renderers, and plugin
  completion.
- `cmd/connectivity/instances` owns instance lifecycle commands, stores,
  renderers, configuration assembly, validation, and instance completion.
- Small shared completion and file helpers stay inside `cmd/connectivity` unless
  both resource packages require them; no unrelated `pkg` refactor is included.

Commands receive the client through an overridable factory. Production uses the
authenticated stack client, while unit tests inject mocks without real network,
authentication, filesystem, or Kubernetes dependencies.

## Completion contract

Completion is part of the feature, not a follow-up:

- plugin arguments complete from `GET /plugins?limit=500`;
- instance arguments complete from `GET /instances?limit=500`;
- `--version` completes from the selected plugin's declared versions;
- `--set` completes schema keys as `KEY=`, includes descriptions, prioritizes
  required keys, and omits keys already supplied;
- after `KEY=@`, completion returns local file and directory candidates while
  preserving the `KEY=@` prefix;
- `--config` and `--env-file` retain normal shell file completion;
- static commands, flags, aliases, and resource names continue to use Cobra's
  generated completion scripts.

Dynamic completion uses a two-second context timeout. Missing profiles,
authentication failures, unsupported Connectivity deployments, and network
errors produce no candidates and no terminal noise. Completion never triggers a
confirmation prompt.

## Rendering and errors

All commands use the existing controller/store/render flow. Plain output uses
`pterm` tables and detail views consistent with nearby `fctl` modules. JSON uses
the global `--output json` path and does not introduce a Connectivity-only
output flag.

Mutating commands use `fctl.WithConfirmFlag` and
`fctl.CheckStackApprobation`. Client-side validation errors name the offending
flag or key. API errors preserve the upstream code, message, details, and HTTP
status. Empty or malformed success responses are treated as errors instead of
being rendered as zero-value resources.

## Testing and verification

Development follows red-green-refactor. Every new production behavior begins
with a focused failing test.

Unit tests use Go's test tooling, `testify`, and mocks:

- client request method, path, query, content type, response decoding, and API
  error decoding through a mocked `RoundTripper`;
- command argument and flag validation through a mocked client interface;
- configuration parsing, source routing, precedence, schema validation, and
  secret redaction;
- plain and JSON rendering;
- every dynamic completion path, including timeout/error silence and `@` file
  candidates;
- root command registration and aliases.

HTTP integration tests may use an in-process `httptest.Server`. Their names use
Given/When/Then and verify complete command-to-API flows without external
services. New handwritten packages must maintain at least 80% statement
coverage.

Verification runs inside the repository's Nix development shell:

```text
nix develop -c just tests
nix develop -c just completions
nix develop -c just pre-commit
```

The last command is the repository's actual precommit recipe and includes tidy,
generated API clients, and linting. Completion scripts are regenerated
explicitly because the precommit recipe does not include them.

## Non-goals

- Removing or deprecating `fctl payments`.
- Reusing `cctl`'s separate contexts, login flow, or output subsystem.
- Exposing Kubernetes objects or kubeconfig directly.
- Adding Connectivity health commands.
- Adding declarative `apply` or full-replacement instance updates.
- Generating and committing an OpenAPI client in this PR.
