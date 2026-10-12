# Connectivity plugin

The local `plugins/connectivity` adapter has been removed. Connectivity service
commands are absent from the current host. Independent plugin distribution and
later host integration are still required; the Auth/Ledger plugin management
commands do not install Connectivity today. Host connection settings remain
available. This removal does not establish a completed migration.

## Historical adapter contract

The following command examples and behavior describe the former embedded
adapter, not currently available commands. That adapter implemented all 14
operations in
[connectivity-api/openapi.yaml](https://github.com/formancehq/connectivity/blob/ce2324887f5b5ec4e3c2ec934ac874656d4c5348/misc/connectivity-api/openapi.yaml),
API version `0.1.0`, pinned to commit `ce2324887f5b5ec4e3c2ec934ac874656d4c5348`.
The repository also provides a generated client under `pkg/client/http`.
The former plugin used the public `pluginsdk/httpclient` adapter to preserve raw
request/response JSON, including merge-patch nulls, unknown fields and exact
numbers, without introducing typed-model defaults such as `replicas`.
It imported no server, Cobra, UI or core packages.
The host owns authentication, endpoints, storage, forms and presentation.

## Preserved host connection settings

```bash
fctl profiles add connectivity-local --auth-mode none \
  --connectivity-url http://localhost:8080
```

`--connectivity-url` / `FCTL_CONNECTIVITY_URL` select a standalone API base URL,
including any deployment prefix. Profiles persist `connectivityURL`. This URL
takes precedence over `--stack-url`, which appends `/api/connectivity`.
Cloud uses the existing scoped Stack client; its gateway must expose Connectivity.
Direct endpoint flags cannot override Cloud routes.
See [profiles and authentication](profiles.md) for profile management, overrides
and session storage.

For OAuth2 credentials, configure `--auth-mode client-credentials`, `--token-url`,
`--client-id`, `FCTL_CLIENT_SECRET` and `--scopes`. With OIDC enabled, the API
accepts `connectivity:read` or `connectivity:write` for reads and requires
`connectivity:write` for writes. Health and info probes are public; query
capabilities use the read authorization boundary.

## Historical catalogue and diagnostics

The former adapter accepted standalone and Cloud targets:

```bash
fctl -p connectivity-local connectivity info
fctl --organization ORG --stack STACK connectivity connectors list
```

```bash
fctl connectivity info
fctl connectivity health
fctl connectivity query capabilities
fctl connectivity connectors list --page-size 30
fctl connectivity connectors facets
fctl connectivity connectors show stripe
fctl connectivity connectors versions list stripe
fctl connectivity connectors versions show stripe stable
```

Version reads accept exact versions and `latest`, `stable`, `rc`, `beta`, `alpha`
aliases. Lists of connectors, versions and instances accept `--page-size` from
1 to 100 (default 15), `--cursor` and `--query`. Facets accept `--query`.
Supply a JSON query object using the fields and operators advertised by
`query capabilities`; the server validates its query grammar.

Responses retain the cursor envelope. Continue with `cursor.next` while
`cursor.hasMore` is true, preserving query and page size. Selectors enumerate
all pages. Versions retain the server's ascending order. JSON retains complete
metadata/spec/status and exact numbers. Tables show names and reconciliation phase.

## Historical instance commands

```bash
fctl connectivity instances create
fctl connectivity instances create --data @instance.json
fctl connectivity instances list
fctl connectivity instances show ingestion
fctl connectivity instances patch ingestion --data '{"suspend":true}'
fctl connectivity instances patch ingestion --data '{"version":null,"channel":"stable"}'
fctl connectivity instances replace ingestion --confirm --data @replacement.json
fctl connectivity instances delete ingestion --confirm
```

Creation opens a form for a DNS-label name, catalogue connector, ledger,
optional version, channel, configuration, start sequence, poll interval and
suspension. The form defaults channel to `stable` and suspension to false.
Explicit bodies bypass forms and gain no defaults. Bodies accept inline JSON,
`@file` and stdin (`--data -`), up to 1 MiB. Creation requires a nonempty connector
and ledger in `spec`. A version pin takes precedence over a channel. Example:

```json
{"name":"ingestion","spec":{"connector":"stripe","ledger":"books","channel":"stable","config":{"env":{"API_KEY":{"secretRef":{"name":"stripe-credentials","key":"api-key"}}}}}}
```

`replace` sends PUT with a required `spec` object and replaces the whole spec.
The server preserves omitted labels/annotations and status. Confirmation is
required; its form asks for the complete replacement spec.

`patch` sends `application/merge-patch+json` directly to **spec**, without a
`spec` wrapper. Omitted fields are preserved; explicit false/empty values stay;
null removes fields. Its form asks for the patch JSON after selecting an instance.
Connector immutability, dynamic configSchema validation, reference resolution
and reconciliation remain server responsibilities. The server owns
`connectivityRef` and `replicas`; use `suspend` to pause ingestion.

Use `secretRef` for sensitive configuration. Unlike schema-managed config,
inline `additionalEnv` and `additionalFiles` values remain on the resource.
The CLI invents no idempotency keys and never retries writes. Read back mutations
and inspect status; write acceptance does not prove healthy ingestion.
Deletion maps the empty 204 response to JSON null.

## Historical verification

These checks describe the adapter before removal. They do not validate a
distributed replacement. The paths below are historical references and are no
longer present in this checkout.

Local fixtures covered all routes/methods, media types, errors, cursor encoding,
int64 precision, size limits, forms, selection and authentication boundaries.
The pinned operation snapshot was `plugins/connectivity/testdata/operations.json`.
An independent consumer verified the public SDK boundary without core or UI
dependencies. A real PTY check covered creation, resource selection, default-No
deletion confirmation, styled forms and plain tables. Creation produced one
write and exact JSON on stdout even with forced terminal colors.
Deployment availability and ingestion require separate live runtime checks.
