# Connectivity commands

`fctl connectivity` manages the published Connector catalogue and the
connections installed in the selected stack. It also reports the Connectivity
API build information. The public API exposes `Connection` resources.

This CLI uses the public Connectivity contract: a connection has a `name`,
`configuration`, and optional `status`. The HTTP routes use
`/connections`. Configuration contains a `connector` name, a `ledger`,
a version or channel selector, and literal `env`/`files` maps. This CLI requires
a Connectivity API exposing that contract.
Read responses may omit connector, ledger or selector for an imported connection;
installing a connection requires a connector, ledger and selector.

## Command tree

```text
fctl connectivity status

fctl connectivity connectors list [--filter KEY=VALUE] [--query JSON]
fctl connectivity connectors show <connector>
fctl connectivity connectors facets [--filter KEY=VALUE] [--query JSON]

fctl connectivity connections list [--connector NAME] [--filter KEY=VALUE]
fctl connectivity connections show <connection>
fctl connectivity connections install <connector> --ledger <ledger>
fctl connectivity connections configure <connection>
fctl connectivity connections suspend <connection> --confirm
fctl connectivity connections unsuspend <connection> --confirm
fctl connectivity connections uninstall <connection>
```

`connection` is the singular alias of `connections`. `connectors ls`, `connectors l`,
`connectors facets` (`facet`, `f`), and `connections ls` (`l`) remain available.
`install` also accepts `create` and `in`; `configure` accepts `config`,
`update`, and `c`. Connection commands use the `connection` and `connections`
JSON envelope keys and the current HTTP field names.

## Service build information

`fctl connectivity status` reads only `GET /api/connectivity/_info` for the selected
stack, using the existing authenticated stack client. It prints Version, Commit
and Build Date. Build Date is the source commit timestamp supplied by the API;
the command does not interpret it as a deployment time or observation freshness.
This reports build metadata, not service health or readiness.

```bash
fctl connectivity status
fctl connectivity status --output json
```

JSON output is a flat build-information object inside the usual data envelope:

```json
{"data":{"version":"v1.2.3","commit":"abc123","buildDate":"2026-10-28T22:29:14Z"}}
```

The three fields must be present strings; empty strings remain valid for builds
without populated metadata. The response may contain additional fields, which
are ignored. The command accepts no positional arguments and returns an error
for unavailable or malformed build information. It does not call a health probe
or use one as a fallback.

## Versions and channels

An install without `--version` or `--channel` explicitly sends `channel:
stable`. `--version` selects a ConnectorVersion by its exact name, including
an administrator-created opaque OCI tag; it does not select a maturity
channel. An opaque tag can follow newly verified artifacts under the operator’s
version rules; selecting its name does not pin its content digest.
`--channel stable|rc|beta|alpha` tracks that channel. Passing both
flags, or an empty selector, is an error.

```bash
fctl connectivity connections install stripe --name stripe-prod --ledger main --channel stable
fctl connectivity connections configure stripe-prod --version v1.0.0
fctl connectivity connections configure stripe-prod --channel beta
```

Changing to `--version` clears the stored channel; changing to `--channel`
clears the stored version. The CLI checks configuration against the selected
version's schema. Channel configuration resolves every catalogue page,
ignores opaque or shortened version candidates, and retains the current
SemVer major and no-downgrade floor. An applied opaque version supplies no
SemVer floor. The server remains authoritative for final validation and
selection during reconciliation. Configuration requires a stored version/channel
selector or an explicit `--version`/`--channel`; it does not infer a selector
from status.

A connection's connector and Ledger destination are immutable once assigned.
`configure --ledger <ledger>` can assign a missing Ledger on an imported
connection or repeat its current value. The API validates the complete resulting
configuration against a published release.
`configure --connector <connector>` can likewise assign a missing connector or
repeat its current value. When configuration inputs are supplied, the requested
connector selects the schema using the stored or explicitly requested version or
channel. Omitted identity flags preserve the stored values; no binding is inferred.
To ingest into another
Ledger, create a separate connection with an explicit name and destination.
No existing history or cursor is copied by that operation.

`--initial-cursor` accepts the connector's opaque cursor encoded as base64.
It is a starting cursor read during registration, not an integer sequence or
a command to rewind a running connection. Changing it on an existing
connection does not prove that ingestion has restarted from that cursor.
The pagination `--cursor` is a separate API-issued token.

## Configuration inputs

`install` and `configure` accept three configuration forms, in precedence
order: `--config` (YAML or JSON), repeatable `--env-file`, then repeatable
`--set KEY=VALUE`. Values may be inline or read locally with `@path`/`@-`.
Configuration files use separate `env` and `files` maps of string values;
flat configuration and flat schemas are unsupported. The CLI sends those maps
as `configuration.env` and `configuration.files`; the file supplied to
`--config` is a configuration input, not the full HTTP request.

```yaml
env:
  API_KEY: example-key
files:
  /etc/plugin/config.yaml: |
    account: example
```

Configuration values are literals. Kubernetes Secret/ConfigMap references,
structured `{value: ...}` entries, file lists and file modes are unsupported.
Supply contents through a local configuration file, dotenv file or
`--set KEY=@path`.
Plain connection output lists configuration keys without values. Debug output
masks literal environment and file contents. Avoid putting confidential literals
directly in shell history.

Malformed YAML/JSON diagnostics identify the input file, report line numbers
when available, and describe the expected structure without echoing input
values.

The CLI derives known keys and definitely-required fields from the Connector
Version JSON Schema, including bounded local `$ref`, `allOf`, `anyOf`, and
`oneOf` composition. Open or pattern-based keys, and constraints that cannot
be proven from CLI input, are sent to Connectivity for authoritative
validation.

`configure` uses JSON merge PATCH. Omitted values, including confidential
fields hidden by GET, are preserved. A `null` entry in `--config` requests
removal of that key; a `null` section requests removal of that section.
Clearing a whole section and then setting entries requires separate configure
requests: a single merge patch cannot express both operations on that section.
Fields marked `format: password` are write-only in API responses. An edit
must not require resupplying an omitted password solely because GET hides it.
The API validates the resulting configuration. Full HTTP PUT replacement is
different: it preserves omitted connector and ledger identities, but clears
omitted mutable fields. Callers must resupply the complete mutable configuration,
including required secrets and a version/channel selector.

## Suspension, resumption and deletion

`connections suspend` and `connections unsuspend` request the desired
ingestion state through the Connectivity API. Both commands require
confirmation, either interactively or with `--confirm`:

```bash
fctl connectivity connections suspend stripe-prod --confirm
fctl connectivity connections unsuspend stripe-prod --confirm
```

A successful command means the request was accepted; reconciliation may still
be in progress. Use `connections show <connection>` to compare desired
`Suspend` with observed lifecycle, ingestion state and `Suspended by` provenance. A Policy
can keep ingestion suspended after the tenant requests resumption. Workload
resources, placement and other administrator settings belong to Policies;
replica count is not a tenant setting.

`uninstall` requests deletion. Kubernetes finalizers may keep the connection
until operator cleanup completes; command success does not prove that the
source has disappeared from Core.

## Filtering, pagination and catalogue metadata

List and facets commands accept repeatable `--filter` expressions:

```bash
fctl connectivity connectors list --filter catalog=ee --filter tags~provider:%
fctl connectivity connections list --filter channel=beta
```

`KEY=VALUE` is an exact match, `KEY!=VALUE` excludes a present matching key,
and `KEY~PATTERN` uses SQL `%` and `_` wildcards. Use `--query '<JSON>'` for
the full API query dialect; it cannot be combined with `--filter`. The CLI
also accepts `--page-size` and the opaque `--cursor` from a previous response.
Shell completion obtains supported keys, values and channels from the API.
Use `lifecycle` and `appliedVersion` for status filters.
Raw `--query` JSON passes through to the
API, including supported `$match`, `$in`, `$exists` and range operators.

The flat Connector response keeps its name, branding, tagline, tags and `latestVersion`
projected by the API for catalogue display. These describe the version
selected for the catalogue card, not necessarily the version installed by a
connection. Full `additionalMetadata` remains available in version detail;
version lists contain summaries. Repository credentials and administrator
configuration are not part of the tenant response.

## Reading status

`configuration.version` or `configuration.channel` is the requested selector;
`status.appliedVersion` identifies the version applied by the control plane.
Lifecycle is the last recorded verdict (`pending`, `active`, `suspended`,
`completed`, `failed`, or `unknown`), not a live health probe.
`status.ingestion.progress.current` is the recorded ingestion progress.
`status.ingestion.lastError` is the last recorded ingestion error and may be
historical after recovery; `connections show` does not fail merely because it
is populated. A recorded service error has a controlled `code` and `message`.
Suspension sources identify a tenant request or a Policy by name.

Authentication failures, including insufficient scope, may return `401`
without a JSON body. Check the selected profile and its permissions instead
of assuming every `401` means an expired session.
