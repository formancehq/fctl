# Connectivity stack gateway facade

This package adapts Connectivity HTTP operations to fctl's existing authenticated
stack client. It appends `/api/connectivity` to the stack URI and uses the supplied
HTTP client and context. Connection routes and the query-capability resource ID are
`connections`; version capabilities use `connectorversions`.

## Public contract

The contract and official SDK come from Connectivity commit
`275545bc44c5e29c9f61edb4ce978898ef2b3455`. Its OpenAPI has SHA-256
`9203f5c4ff1fb1e8225d72a76399c6c7b71330ece6ea3c9e48aa7a0a199f9fd2`
(`misc/connectivity-api/openapi.yaml` and its identical embedded copy).
The importable Go module is `github.com/formancehq/connectivity/pkg/client`;
its HTTP package is `github.com/formancehq/connectivity/pkg/client/http`.
This source pin does not establish a server compatibility range or qualify a
deployed service.

- `GetInfo` delegates to `SDK.Health.Info` for `/_info` and returns local
  `ServiceInfo` with `version`, `commit` and `buildDate`. Presence and string types
  are checked on the raw response; empty strings are valid. Additional fields,
  including debug when present in older replies, are ignored. This operation
  reports build metadata only and calls no health probe.
- Connectors expose flat public fields, starting with `name`. Versions have
  `version`, optional digest, release date and published schema/metadata; they
  do not expose an image reference.
- Connections expose `name`, `configuration` and optional `status`.
  `ConnectionConfiguration` contains `connector`, `ledger`, `version` or
  `channel`, `initialCursor`, `pollInterval`, `suspend`, and literal `env`/`files`
  string maps. `initialCursor` is opaque bytes encoded as base64 on the wire.
  Kubernetes metadata, binding references, storage references and file modes
  are outside this tenant contract.
- POST sends `ConnectionCreate` with `name` and `configuration`. PUT
  sends `ConnectionUpdate` with a complete `configuration`. Create requires
  a connector, ledger and exactly one version/channel selector. PUT requires a
  version/channel selector and complete mutable configuration; omitted connector
  and ledger are preserved. Imported connections may assign missing identities
  after complete server validation. Once set, identities may be repeated but
  cannot be changed or cleared.
- PATCH sends a `configuration` merge patch. Omitted values are preserved;
  `null` deletes a mutable field or individual map entry; it cannot clear an
  assigned connector or ledger. Literal empty strings and
  explicit `suspend: false` survive encoding. The client rejects administrative
  fields, reference objects and non-string values. The service validates the
  merged selector and immutable fields.
- Read responses omit confidential password values and reference-backed
  entries. Do not infer an omitted value is empty. PUT must provide complete
  configuration, including required secrets; use PATCH to preserve omissions.
  Imported connections without connector or ledger bindings are readable; absent
  identities remain absent in JSON. Create still requires both identities.
- Optional status reports recorded `appliedVersion`, `lifecycle`, ingestion
  state/progress/lastError, service error and request/policy suspension sources.
  The service error is a sanitized projection; ingestion `lastError` is the raw
  last recorded ingestion diagnostic and can remain after recovery.
  Progress is a plugin-defined integer, not a universal sequence or percentage.
  Absent progress differs from a recorded zero. These fields do not prove live
  worker health, observation freshness or an active error.

Success responses require an object root. Lists require non-null `cursor` and
`cursor.data`; malformed resources reject the whole page. Resource identities,
populated connection ledger, status, and progress are validated. Empty lists and
absent status remain valid. Structured API errors retain HTTP status, code,
message and details; empty or non-JSON errors retain status text, including 401.

## Generated SDK boundary

The facade in `client.go` delegates service build information, catalogue, version,
capability and Connection HTTP operations to the official SDK. It preserves
fctl's `Client`,
list wrappers, byte cursor representation and `APIError` conventions.
`cmd/connectivity/internal/client.go` supplies the selected stack URI and
authenticated `StackClients.HTTPClient`.

Each invocation constructs an SDK client around the supplied HTTP client and
caller context, without adding authentication, timeout or retry policy. The SDK
owns routes and request serialization. This SDK revision substitutes path
parameters verbatim, so the facade escapes each name/version argument before
calling an operation. Validated facade write DTOs convert to
SDK models; the SDK's nullable PATCH model preserves omitted fields, explicit
nulls, false and empty string values without a GET/merge step. A request decorator
removes the SDK's injected `pageSize=15` only when no size was requested.

The invocation captures the original response bytes and closes the network body
before the SDK reads its replay. Strict facade validation runs on those original
bytes, because generated models can erase missing required values into zero.
The SDK models JSON HTTP errors across 4XX/5XX with one generic Error; the
facade retains HTTP status, code, message and details, including 429 and 599.
Normalization uses the original status/body even if SDK decoding fails;
empty or malformed errors retain status text. Successful response StatusError
records remain separate from HTTP errors. Stream read failures propagate
without returning a partially decoded result. A successful DELETE closes its
body without reading it. Response capture is invocation-local, so concurrent
calls share no mutable response state. As before, JSON success decoding does
not depend on the response Content-Type header.

Tests import no Connectivity server implementation. Consumer fixtures verify
HTTP behavior with the actual SDK dependency, not a live Connectivity backend.

## Verification

Source builds require read access to the private Connectivity repository at the
pinned revision. Configure Git authentication through your existing credential
setup and set `GOPRIVATE=github.com/formancehq/connectivity` so Go resolves this
module without the public proxy/checksum service. CI uses the shared workflows'
private-repository token input; successful access still needs CI verification.

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
