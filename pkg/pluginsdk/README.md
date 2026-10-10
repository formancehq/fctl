# fctl plugin SDK

This independent Go module defines the public fctl v4 plugin contract. It has
no dependency on Cobra, terminal forms, profiles, or the core's authentication.
Go 1.26 or newer is required.

An embedded or external plugin implements `GetManifest` and `Execute` and
exposes a factory with the signature `func(*http.Client) pluginsdk.Plugin`.
`GetManifest` must work with a nil HTTP client and without service requests.
The manifest includes declarative forms; fctl owns their presentation.

## External executable

```go
func main() {
    transport.Serve(ledger.New)
}
```

The host uses `transport.Open(ctx, absoluteBinaryPath, httpClient, endpoint)`
and defers `Close`. For metadata only, it can supply a nil client and an empty
endpoint. Execute requires an injected client and endpoint. Its request may
omit Endpoint; a supplied Endpoint must match the host's endpoint.

The gRPC runtime follows the process and broker design of fctl PR #126 while
retaining the current v4 contract. The HTTP callback uses `GRPCBroker` and
local mutual TLS. Only the host HTTP client contacts the service. Its
authentication, token refresh, request debugging, timeout and cookie jar
continue to apply. Every request and redirect must stay within the endpoint's
origin and path boundary. Dot segments and encoded traversal are rejected.
Credential headers supplied by the plugin are rejected; credential response
headers are removed. Authentication credentials must never be placed in
`ExecuteRequest.Context`.

The child receives no command-line arguments beyond its executable path and
does not inherit the host's environment. Go-plugin supplies only its protocol
and public TLS handshake values; Windows also receives SYSTEMROOT. The runtime
discards plugin stdout and stderr. Ordinary HTTP transport failures expose a
generic message rather than host paths or credentials.

Service JSON is transferred as bytes, preserving whitespace and integer
precision. Structured `httpclient.Error` fields and partial response data are
preserved. Request bodies are limited to 4 MiB, request metadata and manifests
to 1 MiB, HTTP headers to 64 KiB, and total result data plus structured error
to 32 MiB. gRPC messages are limited to 48 MiB to include base64 framing. The
runtime performs no automatic HTTP or gRPC retries.

Canceling the Open context or an in-flight method terminates the executable;
it cannot be reused afterward. Close is idempotent and waits for process
cleanup. Only trusted, verified executables should be opened: a separate
process is not an operating-system sandbox. Artifact discovery, version
selection, digest verification, and installation belong to the host.

## Validation

From this module directory:

```sh
go test -race ./...
nix develop --impure --command golangci-lint run --config ../../.golangci.yml ./...
```

The transport suite launches the test binary as an actual plugin, with no
helper arguments or inherited environment, and separately exercises gRPC
servers in-process for coverage. It checks HTTP ownership, credential
isolation, endpoint boundaries, redirects, exact JSON, errors and partial
results, limits, concurrent callbacks, process crashes, and cancellation.

### Optional aliases and file effects

`CommandSpec.Aliases` gives alternative command names. `FindCommand` accepts
them and `NormalizeRequest` returns a canonical command path. Hosts validate
alias collisions when registering a manifest.

`CommandSpec.Files` declares input argument/flag and output flag effects.
The host reads a local file, stdin (`-`) or an explicit HTTPS source into Body;
plugins do not open these sources. `ReadFormat` selects JSON, YAML-to-JSON or
raw text encoded as a JSON string. Inputs are bounded to 4 MiB. `WriteFormat`
selects JSON, NDJSON or YAML. `FormatFlag` can override it; `yml` means YAML.
File writes are atomic and private, and execution errors do not create exports.
Direct SDK callers supply already-read Body and consume Data as before.
