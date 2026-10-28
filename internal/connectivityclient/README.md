# Connectivity stack gateway facade

This package adapts Connectivity HTTP operations to fctl's existing authenticated
stack client. It appends `/api/connectivity` to the stack URI and uses the supplied
HTTP client and context. Instance routes and the query-capability resource ID are
`connector-instances`; version capabilities use `connectorversions`.

## Public contract

The contract pin is Connectivity
`761313ebf3b456cc47aa2ef16d3502ab121f1522`:
[OpenAPI](https://github.com/formancehq/connectivity/blob/761313ebf3b456cc47aa2ef16d3502ab121f1522/misc/connectivity-api/openapi.yaml)
and [generated SDK models](https://github.com/formancehq/connectivity/tree/761313ebf3b456cc47aa2ef16d3502ab121f1522/pkg/client/http/models/components).
This pin identifies the inspected contract, not a broader server compatibility
range or a deployed-server qualification.

- Connectors expose flat public fields, starting with `name`. Versions have
  `version`, optional digest, release date and published schema/metadata; they
  do not expose an image reference.
- Instances expose `name`, `configuration` and optional `status`.
  `ConnectorInstanceConfiguration` contains `connector`, `ledger`, `version` or
  `channel`, `initialCursor`, `pollInterval`, `suspend`, and literal `env`/`files`
  string maps. `initialCursor` is opaque bytes encoded as base64 on the wire.
  Kubernetes metadata, binding references, storage references and file modes
  are outside this tenant contract.
- POST sends `ConnectorInstanceCreate` with `name` and `configuration`. PUT
  sends `ConnectorInstanceUpdate` with a complete `configuration`. Both require
  a connector, ledger and exactly one version/channel selector. Changing the
  connector or ledger remains a server-validated error.
- PATCH sends a `configuration` merge patch. Omitted values are preserved;
  `null` deletes a field or individual map entry. Literal empty strings and
  explicit `suspend: false` survive encoding. The client rejects administrative
  fields, reference objects and non-string values. The service validates the
  merged selector and immutable fields.
- Read responses omit confidential password values and reference-backed
  entries. Do not infer an omitted value is empty. PUT must provide complete
  configuration, including required secrets; use PATCH to preserve omissions.
  Stored instances without a connector binding are readable, as allowed by the
  contract, but cannot be submitted through POST or PUT without a connector.
- Optional status reports recorded `appliedVersion`, `lifecycle`, ingestion
  state/progress/lastError, sanitized error and request/policy suspension sources.
  Progress is a plugin-defined integer, not a universal sequence or percentage.
  Absent progress differs from a recorded zero. These fields do not prove live
  worker health, observation freshness or an active error.

Success responses require an object root. Lists require non-null `cursor` and
`cursor.data`; malformed resources reject the whole page. Resource identities,
instance ledger, populated status, and progress are validated. Empty lists and
absent status remain valid. Structured API errors retain HTTP status, code,
message and details; empty or non-JSON errors retain status text, including 401.

## Generated SDK boundary

The handwritten facade in `client.go` preserves fctl's `Client`, list wrappers
and `APIError` conventions. `cmd/connectivity/internal/client.go` constructs it
with the selected stack URI and authenticated `StackClients.HTTPClient`. The pinned official SDK lives at
`github.com/formancehq/connectivity/pkg/client` and represents `initialCursor`
as a base64 string; this facade exposes bytes. Tests inspect the public shape
without importing the server or the generated SDK.

## Verification

From the repository root, use the pinned toolchain:

```sh
go test ./internal/connectivityclient
go test -race ./internal/connectivityclient
```

Transport tests cover escaped paths, query options, response validation and
errors. Portable Given/When/Then fixtures use `httptest` to exercise actual
POST/PUT/PATCH/GET requests, literal maps, cursor base64, selector switching,
confidential-value omission, status decoding and null deletion markers. These
fixtures are bounded consumer tests, not executable proof against a running
Connectivity server or the full CLI.
