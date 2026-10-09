# Plugin contract

Cloud, Auth and Ledger implement `pkg/pluginsdk.Plugin`. Auth is loaded only as
an external executable; Cloud and Ledger have embedded providers. The public SDK
is a separate Go module, `github.com/formancehq/fctl/pkg/pluginsdk`. It has no Cobra,
profile-store or core authentication dependency. Each plugin exposes two methods:

- `GetManifest` describes its command tree, arguments, typed flags and declarative
  interactive inputs.
- `Execute` accepts a command path, arguments, resolved flags, changed flags,
  a JSON body, a resolved service endpoint and non-secret host context. It returns
  raw JSON and an error.

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

## Service selection and gates

`Manifest.Service` identifies the default host connection boundary.
`CommandSpec.Service` overrides it for a command and its descendants; the nearest
override wins. `pluginsdk.CommandService` resolves this value. Cloud normally
uses `cloud` (Membership), while `cloud apps` declares `cloud-apps` (Deploy).
The host resolves the authenticated client for that service, then injects its
endpoint and context into the request.

`FlagSpec.RequireTrue` applies only to boolean flags and requires the resolved
value `true`. Apps uses it for its inherited `--experimental` flag. An omitted
default-false flag or `--experimental=false` is rejected before body-file access,
connection resolution or plugin construction. Destructive commands separately
declare `CommandSpec.Confirm` and require `--confirm=true`.

`ExecuteRequest.Context` is a copied `map[string]string` of non-secret host
metadata. The Cloud host supplies selected and available organization/stack IDs;
the application host supplies organization and application alias. Tokens,
credentials and profile paths do not belong in this map. Authentication stays
in the injected HTTP transport.

See [interactive input](interaction.md) for terminal fields, choice discovery and
automation. Input metadata does not relax direct SDK validation.

## Host ownership

The core registers plugin factories and builds Cobra commands from their
manifests. It owns profiles, authentication, private token persistence, signal
cancellation, reading `--data @file` and stdin bodies, and [output rendering](output.md). It
resolves an authenticated HTTP client and the endpoint before invoking a plugin.
The plugin owns service routes, payload schemas, idempotency headers and business
validation. Both the CLI adapter and direct SDK execution validate arguments and
flags. Explicit changed flags preserve the difference between an omitted boolean
and `--flag=false`.

Plugins receive an HTTP client through their constructor. This is the embedded
adapter's dependency injection point; plugin code does not obtain credentials
or read saved profiles. The adapter validates and reads the body, resolves
the declared service once, copies endpoint/context, executes the plugin and
renders any returned data, including partial results accompanied by an error.
Service command trees and schemas remain in plugins rather than in the host.
Apps file upload/download operations additionally own their declared `--path`
and `--out` effects; they are not connection configuration. The shared public
HTTP client performs requests without automatic mutation retries and preserves
JSON numbers.

Cloud, Ledger and Connectivity remain available as embedded defaults. Ledger
can replace its embedded command tree with a cached external manifest. Auth is
external-only: the host loads its cached manifest and executable, or discovers
and installs an exact-version release before execution. There is no embedded
Auth fallback. Both external providers use the same two-method SDK contract,
forms, body validation, authentication and output renderer. Exactly one provider
owns each service command root.

External executables use the SDK's `transport.Serve` and the host's
`transport.Open`. The gRPC broker supplies a scoped HTTP callback; the host
retains authentication credentials and follows requests through its own HTTP
client. Plugin requests and redirects must stay within the resolved endpoint's
origin and path prefix. The child inherits no host environment or credential
arguments. A separate process is not an operating-system sandbox: only trusted
executables should be installed.

The host checks the binary checksum and compares its runtime manifest with the
installed manifest before execution. It also checks the exact selected service
version before service operations. Canceling an RPC closes the plugin process.
Help and completion use cached metadata, without spawning a plugin or calling
the service or catalogue. On a fresh target, `fctl auth --help` shows only a
placeholder directing the caller to `plugins sync --service auth` or
`plugins install --service auth`. Full Auth help and completion become available
offline after preparation of that target.

See [plugin distribution](plugin-distribution.md) for Auth and Ledger, exact
version selection, public OCI packages, target locks and local testing.

## Product-owned Auth plugin

The independent module `github.com/formancehq/auth/misc/fctl-plugin` owns Auth
commands, API validation and declarative forms in the Auth repository. fctl has
no `plugins/auth` adapter and no compile-time dependency on the Auth product
module (`github.com/formancehq/auth/misc/fctl-plugin`).
The product module exposes `New` for direct SDK callers and `NewVersion` for
exact-version native executables. It depends on the public plugin SDK and
generated Auth client, without fctl core, Cobra, profile storage or terminal
libraries. Host authentication remains in fctl and does not require embedding
Auth commands.

Auth and Ledger external providers use the same generic host loader. The host
accepts only the selected service's command root and connection boundary,
including descendant service overrides. Locks are keyed by target and service.
