# Connectivity plugin

The external product-owned module
[`github.com/formancehq/connectivity/misc/fctl-plugin`](https://github.com/formancehq/connectivity/tree/main/misc/fctl-plugin)
implements all 14 Connectivity API operations. It is installed through the
shared loader; fctl has no Go dependency on the product module and no embedded
Connectivity fallback. The product owns forms, payload validation and HTTP
mapping; fctl owns credentials, terminal interaction and cached manifests.

## Profiles

```bash
fctl profiles add connectivity-local --auth-mode none \
  --connectivity-url http://localhost:8080
fctl -p connectivity-local connectivity info
fctl --organization ORG --stack STACK connectivity connectors list
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

## Catalogue and diagnostics

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

## Instances

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

## Verification

Local fixtures cover all routes/methods, media types, errors, cursor encoding,
int64 precision, size limits, forms, selection and authentication boundaries.
The operation snapshot and current API contract tests live in the product module.
An independent consumer verifies the public SDK boundary without core or UI
dependencies. A real PTY check covers creation, resource selection, default-No
deletion confirmation, styled forms and plain tables. Creation produces one
write and exact JSON on stdout even with forced terminal colors.
Deployment availability and ingestion require separate live runtime checks.

## Install the exact product executable

```sh
fctl plugins install --service connectivity --binary ./fctl-plugin-connectivity --profile connectivity-local
fctl plugins sync --service connectivity --profile connectivity-local
fctl plugins show --service connectivity --profile connectivity-local
```

Without an installed matching executable, Connectivity commands require plugin
installation. Help/completion use cached metadata without launching a process or
contacting the service. Each execution checks `/_info` again and rejects version
drift before starting the plugin. Patch input remains a spec-only JSON object;
the product wraps it as `{"spec":...}` for the API. Read-only replicas and
connectivityRef are rejected locally; use suspend.
