# Connectivity stack gateway facade

This package adapts Connectivity HTTP operations for fctl's existing stack
client. It appends `/api/connectivity` to the stack URI and uses the supplied
HTTP client, including its authentication and transport configuration. The
public CLI calls these resources connections; API routes and query capability
resource IDs remain `connectorinstances`.

The facade keeps a stable CLI-facing `Client`, models, and `APIError` surface:

- Success responses must have an object root. Lists require a non-null
  `cursor` object and `cursor.data` array; individual resources require their
  identifying fields. Valid empty lists remain readable.
- Stored legacy instances without `spec.connectorRef` remain readable.
  Binding, ledger, and version/channel write validation belongs to the server.
- Structured API errors retain status, code, message, and details. An empty
  error body, including HTTP 401, retains HTTP status and status text.
- Typed create fields omit unset optional values and preserve explicit boolean
  false and empty inline strings. Merge patches use a map to preserve the
  distinction between omission, explicit `null`, and empty arrays or objects.

## Instance wire contract

The audited server contract is Connectivity revision
`4084c5667ca17ce1ff3e0a5d6364ce5f3072e77f`, specifically
[`openapi.yaml`](https://github.com/formancehq/connectivity/blob/4084c5667ca17ce1ff3e0a5d6364ce5f3072e77f/misc/connectivity-api/openapi.yaml)
and [`mapping.go`](https://github.com/formancehq/connectivity/blob/4084c5667ca17ce1ff3e0a5d6364ce5f3072e77f/misc/connectivity-api/internal/server/mapping.go).
This audit pin identifies the tested contract; it does not establish a broader
server compatibility range.

`ConnectorInstanceSpec` contains only `connectorRef`, `version`, `channel`,
`connectivityRef`, `ledger`, `initialCursor`, `pollInterval`, `suspend`, `env`,
and `files`. Both references are objects with a `name`. `initialCursor` is
opaque `[]byte`, encoded as base64 by JSON. `ConnectorInstanceConfig` is an
assembly helper; callers place its `Env` and `Files` directly in the spec or
merge patch. `ConnectorName()` tolerates a missing reference, and
`Configuration()` returns an Env/Files helper sharing those values.

`connector`, `config`, `startSequence`, and `replicas` are retired wire fields.
No client-side rewriting or installable-artifact fallback restores them.

## Generated client adoption gap

Accepted [RFC-0012](https://github.com/formancehq/internal-rfcs/blob/391d9359c65edf2a4f196ad5689ddb8b0206b5aa/rfcs/0012-module-api-transport-and-client-distribution.md)
makes the official generated Go package the default and requires a documented
requirement for a custom client. The stack routing, strict response envelopes,
stable CLI-facing models/error handling, and omission/null controls above are
this existing facade's requirements. They do not prove that the generated SDK
cannot support those requirements behind an adapter.

The official module `github.com/formancehq/connectivity/pkg/client` exists at
the audited server revision, declares Go 1.22, and exposes HTTP types under
`http/models/components`. Its generated spec exposes the same tenant fields;
its `InitialCursor` is `*string` containing base64, while this facade exposes
bytes. fctl currently does not depend on that module, and these tests do not
import its types or execute its operations.

This bounded adaptation does not decide to replace the SDK, establish an
approved RFC exception, or claim generated-client conformance. The fctl
integration owner must assess using an immutable normal Go dependency behind
the facade, verify cursor conversion and omission/null behavior, and add
executable contract parity checks before claiming adoption. No generated
source or contract is copied into this package.

## Verification

From the repository root:

```sh
nix develop --command go test ./internal/connectivityclient
```

Unit tests use a fake HTTP transport. Given/When/Then integration tests use an
isolated HTTP server to check POST/PATCH wire bodies, reference shapes, cursor
base64, version/channel selectors, configuration values, deletion markers,
empty file arrays, and decoded metadata/status projections. These tests do
not qualify a deployed Connectivity server or the complete CLI integration.
