---
name: formance-fctl-plugin
description: "Create or extend embedded Go service plugins in fctl v4: manifests, declarative forms, API operations and contract tests. Use for plugin development rather than operating the CLI or designing an external plugin runtime."
---

# fctl plugin development

Build service commands that work through both the CLI and a standalone SDK
caller. Keep service behavior in the plugin and terminal/authentication behavior
in the host. Embedded plugins are the current delivery model; their public
contract also permits a future external adapter.

## Establish the contract

Locate the requested fctl checkout and read its `go.mod`, `docs/plugins.md`,
`pkg/pluginsdk/sdk.go` and `pkg/pluginsdk/interaction.go`. These files take
precedence over assumptions about older fctl releases. If the checkout lacks
this SDK, establish the intended version before creating another architecture.

Read the service's published client or API definitions at the requested branch
or version. Use the small Auth plugin for SDK-backed operations, Ledger for
precise JSON and keyset pagination, and Cloud for multi-service commands. Pick
the relevant example rather than copying the whole tree. For Ledger, consult
the current source contract in `docs/validation.md` and the requested release
branch; a deployed beta can differ from that contract.

Map each requested command to its method, versioned route, payload, response,
pagination, and failure behavior. Distinguish missing CLI behavior from an API
capability absent in the deployed service.

## Implement the plugin

Create or extend `plugins/<service>`. Expose `New(*http.Client) pluginsdk.Plugin`,
`GetManifest(context.Context)` and `Execute(context.Context, ExecuteRequest)`.
Metadata must be available with a nil client and without network access.

Normalize every execution with `pluginsdk.NormalizeRequest`, then validate
identifiers and the service payload before issuing a request. Preserve
`ChangedFlags` when omitted values and explicit false/empty values differ.
Return a useful error when execution lacks an injected HTTP client.

Consume the injected client, endpoint and non-secret context. Use the published
service client when it implements the needed contract, or the public
`pkg/pluginsdk/httpclient` boundary for missing operations. Keep authentication
and endpoint selection in the host. Plugin packages depend on public SDK/client
packages, not Cobra, UI libraries, fctl `internal` packages or server commands.

Preserve `json.RawMessage` or `json.Number` for quantities and identifiers.
Pass raw IDs to `httpclient.Path`; it escapes each segment once, including a
slash or literal percent sign. Avoid pre-escaping these inputs. Validate empty
IDs and dot-segment IDs before building service routes. Keep
mutation retries explicit; the shared HTTP boundary performs one attempt.
Return partial response data alongside an error when the API contract requires it.

## Describe the CLI

Declare commands, argument counts, flags and help in the manifest. Use inherited
`Target` for identity/organization/stack selection, and `Service` overrides for
different host connection boundaries. Declare destructive operations with
`Confirm: true` and a boolean `confirm` flag defaulting to false.

For missing-input forms, resource selectors or a payload editor, read
[declarative inputs](references/inputs.md). Put serializable input metadata on
runnable leaves; the core renders the UI and validates the accepted values.

For authorized embedded integration, register the factory in the existing
`cmd/root.go` registry and give its root command the appropriate help group.
For an isolated evaluation, use a temporary host harness and leave the requested
checkout unchanged. A new service may also need host
endpoint, scope and connection configuration support: inspect the current
resolver rather than assuming a new manifest service is automatically usable.
Keep these changes at the connection boundary; routes and payloads remain in
the plugin. Existing service extensions usually need no core changes.

## Verify and deliver

Read [contract and CLI checks](references/verification.md) for the checks that
apply to the new behavior. A plugin is ready when its API behavior passes local
fixtures, its manifest registers, a standalone caller runs without core/UI
dependencies, and the requested CLI path works in interactive and explicit modes.

Use authorized disposable resources for live verification when requested, then
read back changes and clean up owned fixtures. Report fixture coverage, real
service results, unsupported backend behavior and unverified paths separately.
