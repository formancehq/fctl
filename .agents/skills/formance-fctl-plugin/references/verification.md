# Verify a plugin independently and through fctl

Use local HTTP fixtures for API behavior before relying on a real stack. Read
the checkout's test conventions and `justfile`; use its pinned Go/Nix toolchain.

## API and manifest evidence

- Call `GetManifest` with a nil HTTP client and marshal/unmarshal the manifest.
  Register it through the existing registry tests to catch invalid bindings,
  command paths, choice sources and inherited flags.
- Execute each added leaf through the public SDK against `httptest.Server`.
  Check method, escaped route, query, body and relevant headers. Verify wrong
  argument counts, unknown flags, invalid payloads and refused confirmations
  fail before HTTP. Include malformed/error responses relevant to the API.
- Exercise omitted update options versus explicit false/empty values when
  supported. Verify `9007199254740993` survives a request and a response exactly
  when the operation carries quantities, metadata or numeric IDs.
- Check cancellation and write count for mutations. Add idempotency/partial
  result checks when those behaviors belong to the service contract.

Use the public API, not implementation-mirroring assertions. A standalone
consumer can use a temporary module with a local `replace` for
the public SDK and product plugin modules (or `github.com/formancehq/fctl/v4`
for an embedded service), then import the plugin and call `New`,
`GetManifest` and `Execute`. Use `GOWORK=off` so workspace settings do not hide
dependencies. Inspect its dependency closure for Cobra, pflag, UI and fctl core
packages; none belongs to the plugin's public execution path. Do not treat the
host CLI's own dependency closure as the plugin closure.

Registry and Runner integration checks belong in the fctl module, because those
host packages are `internal`. For an external plugin module, keep its SDK tests
independent and place host integration tests in fctl or an isolated host checkout.
The standalone consumer must exercise the public dependency boundary separately.

## Forms and scripting

Use the core's injected `interactive.Runner` in adapter fixtures to cover:

- Missing values bind to the expected argument/flag/body/context without an
  extra write; cancellation and default-No confirmation perform no mutation.
- Explicit values and a complete `Body` skip forms, including inline JSON,
  `@file` and stdin input. Disabled interaction returns an actionable error.
- Selectors return actual IDs, handle empty lists and use GET-only discovery.
  If paginated, test a second page and repeated/missing continuations; if filtered,
  test a full ineligible page followed by an eligible page.

Use a real PTY when adding a new interactive path. Check one complete form,
resource selection and cancellation at ordinary terminal dimensions. Capture
stdout separately from stderr and parse `-o json` even with forced color. Check
human tables with `NO_COLOR`. Runner fixtures cannot verify visual rendering or
actual keyboard navigation.

## Repository checks and live work

For authorized new plugins or host integration, run the repository checks,
for example:

```bash
nix develop --impure --command just pc
nix develop --impure --command just tests
```

Inspect changes from formatter/generator recipes before committing. An isolated
SDK consumer needs its own `go test -race ./...`; the fctl suite does not discover
tests in an external module automatically. When adding a host service boundary,
cover Cloud, anonymous and client-credentials modes in connection fixtures.

For a read-only or temporary evaluation, run formatting, race tests, vet and
dependency checks only in the temporary module or host copy. `just pc` writes
module, generated, lint-fix and completion files; it belongs in the writable
integration checkout, not a read-only source checkout. Report this smaller scope.

When live testing is in scope, use the selected sandbox and newly owned fixtures.
Read back mutations, retain user-owned resources and verify cleanup. Report the
selected stack version and actual service version separately. Deployed betas
can lack fields present in a release branch; fixture success proves the client
contract, while live readback proves deployed behavior. CLI work does not grant
permission to publish releases, send invitations or modify unrelated resources.
