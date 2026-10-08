# Embedded plugin contract

Auth and Ledger implement `pkg/pluginsdk.Plugin`. The public SDK has no Cobra,
profile-store or core authentication dependency. Each plugin exposes two methods:

- `GetManifest` describes its command tree, arguments and typed flags.
- `Execute` accepts a command path, arguments, resolved flags, changed flags,
  a JSON body and a resolved service endpoint. It returns raw JSON and an error.

The manifest and execution messages are JSON-serializable. Amounts remain raw
JSON integers. A response can accompany an error so bulk partial results are
preserved. Protocol version 1 identifies this contract; it is inspired by the
manifest/execution split in Geoffrey's [proposal](https://github.com/formancehq/fctl/pull/126).
It is not wire-compatible with that draft's protobuf messages.

An independent Go caller can use the same plugin without constructing a CLI:

```go
p := ledger.New(authenticatedHTTPClient)
manifest, err := p.GetManifest(ctx)
// Handle err before using manifest.
response, err := p.Execute(ctx, pluginsdk.ExecuteRequest{
    CommandPath: []string{manifest.Name, "list"},
    Endpoint:    "https://stack.example/api/ledger",
})
// Inspect response.Data even when err is non-nil for bulk operations.
```

For writes, supply already-read JSON in `Body`. Direct SDK callers do not need
CLI-only values such as `--data @file`; `NormalizeRequest` validates the payload
boundary and resolved flags.

The core registers plugin factories and builds Cobra commands from their
manifests. It owns profiles, authentication, private token persistence, signal
cancellation, reading `@file` and stdin bodies, and writing JSON output. It
resolves an authenticated HTTP client and the endpoint before invoking a plugin.
The plugin owns service routes, payload schemas, idempotency headers and business
validation. Both the CLI adapter and direct SDK execution validate arguments and
flags. Explicit changed flags preserve the difference between an omitted boolean
and `--flag=false`.

Plugins receive an HTTP client through their constructor. This is the embedded
adapter's dependency injection point; plugin code does not obtain credentials,
read configuration or write CLI output. The shared public HTTP client performs
requests without automatic mutation retries and preserves JSON numbers.

Auth and Ledger remain compiled into the executable. There is no installation,
registry download, external process launcher or gRPC transport in this change.
An external adapter must implement transport negotiation and authenticated HTTP
context delivery using this public contract. That adapter will not require the
service implementations to import Cobra or core configuration.
