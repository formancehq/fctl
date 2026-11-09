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
fctl connectivity
├── server-infos
├── connectors
│   ├── list
│   └── show <connector>
└── connections
    ├── list
    ├── show <connection>
    ├── install <connector> --ledger <ledger>
    ├── configure <connection>
    ├── suspend <connection> --confirm
    ├── unsuspend <connection> --confirm
    └── uninstall <connection>
```

`connection` is the singular alias of `connections`. `connectors ls`, `connectors l`,
and `connections ls` (`l`) remain available.
`server-infos` also accepts `si`. `install` also accepts `create` and `in`.
The `config` group (alias `configuration`) is hidden from help and completion
because `configure` covers it; `config update` keeps working. `connections configure <connection>`, `connections update <connection>`,
and `connections c <connection>` remain compatible with earlier CLI builds.
`config` is now a command group: replace the previous `connections config NAME`
alias with `connections config update NAME` or `connections configure NAME`.
All update routes use the same controller, input validation, completion and
JSON merge-PATCH behavior. A connection named `update` is addressed by
`config update update` or `configure update`. Connection commands use the `connection` and `connections`
JSON envelope keys and the current HTTP field names.

## Command packages

The command tree lives entirely under `cmd/connectivity`: `connectors` owns
catalogue commands, `connections` owns lifecycle commands, and
`connections/configuration` owns configuration grouping and schema-aware input
parsing. Commands call the generated Stack SDK in `pkg/client` directly.
`internal` holds completion helpers and the merge-patch body adaptation.

## Service build information

`fctl connectivity server-infos` reads only `GET /api/connectivity/_info` for the selected
stack, using the existing authenticated stack client. It prints Version, Commit
and Build Date. Build Date is the source commit timestamp supplied by the API;
the command does not interpret it as a deployment time or observation freshness.
This reports build metadata, not service health or readiness.

```bash
fctl connectivity server-infos
fctl connectivity server-infos --output json
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
fctl connectivity connections config update stripe-prod --version v1.0.0
fctl connectivity connections config update stripe-prod --channel beta
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

A connection's connector and Ledger destination are set at installation and
are immutable once assigned. `configure` accepts neither `--connector` nor
`--ledger`: it sends only mutable fields in a merge PATCH. The API rejects any
connector or ledger field in PATCH with `400 unsupported_field`, even when the
value matches the stored binding or is `null`. To use another connector or
Ledger, install a separate connection with an explicit name and destination.
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
Plain connection output displays configuration values returned by the API.

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
`Suspend` with observed lifecycle, ingestion state and `Suspended By` provenance. A Policy
can keep ingestion suspended after the tenant requests resumption. Workload
resources, placement and other administrator settings belong to Policies;
replica count is not a tenant setting.

`uninstall` requests deletion. Kubernetes finalizers may keep the connection
until operator cleanup completes; command success does not prove that the
source has disappeared from Core.

`suspend` and `unsuspend` JSON output includes the `connection` returned by the
API next to `name` and `suspend`.

## Terminal output

Plain output fits the width of the terminal it is written to. Columns are never
hidden. When a table is too wide, the widest column allowed to shrink loses one
character at a time: identifiers such as names, keys and versions are never
cut, catalogue text ends with `…`, and descriptions and long values wrap onto
more lines. When the table still does not fit at those minimum widths, each row
prints as a stacked block of labelled values. When output is piped or
redirected, nothing is cut or wrapped. `--output json` always returns the
complete API models.

`connectors list` displays Name, Latest Version and Tagline. Display names
repeat the connector name, so they appear only in `connectors show`. Full tag
lists remain available through `connectors show`, JSON output and filter
completion.

`connections list` displays Name, Connector, Ledger, Target, Applied and
Status. Target is the requested version or channel. Status is the lifecycle;
the ingestion state is appended only when it differs from the expected one
(`running` while active, `paused` while suspended, `finished` once completed).
Progress and last-error details are available through `connections show` and
JSON output.

`connections show` summarizes multi-line file values by their first line and
line count, and long initial cursors by their first characters and length.

`connectors show` groups configuration fields by the schema's
`presentation.groups`, in the publisher's order, and marks advanced groups.
Each field shows its key, whether it is required, its type (`secret` for
password fields) and the first sentence of its description followed by its
default. Schema `requirements` are listed under Rules.

A list prints `Next page: --cursor <token>` when more results exist. The token
is never shortened.

## Filtering, pagination and catalogue metadata

List commands accept repeatable `--filter` expressions:

```bash
fctl connectivity connectors list --filter catalog=ee --filter tags~provider:%
fctl connectivity connections list --filter channel=beta
```

`KEY=VALUE` is an exact match, `KEY!=VALUE` excludes a present matching key,
and `KEY~PATTERN` uses SQL `%` and `_` wildcards. Use `--query '<JSON>'` for
the full API query dialect; it cannot be combined with `--filter`. The CLI
also accepts `--page-size` and the opaque `--cursor` from a previous response.
Shell completion obtains supported keys, values and channels from the API.
Connector facets supply tag suggestions for `connectors list --filter`, and the
published connectors supply `catalog` values. Both counts apply the `--filter`
expressions already typed on the command line, so `--filter tags=provider:psp
--filter tags=region:<TAB>` counts only PSP connectors. Repeated `--filter`
expressions combine with AND; use `--query` with `$or` or `$in` for OR.
Use `lifecycle` and `appliedVersion` for status filters.
Raw `--query` JSON passes through to the
API, including supported `$match`, `$in`, `$exists` and range operators.

The flat Connector response keeps its name, branding, tagline, tags and `latestVersion`
projected by the API for catalogue display. `connectors show` fetches the schema
using that exact `latestVersion`; it does not resolve a `latest` alias or infer
a version from the list. If the field is absent or empty, it skips the schema
request and still displays the number of available versions. Plain output shows
this count and the latest version's release date and shortened digest instead
of the full version history; JSON output retains all version summaries. These describe the version
selected for the catalogue card, not necessarily the version installed by a
connection. Full `additionalMetadata` remains available in version detail;
version lists contain summaries. Repository credentials and administrator
configuration are not part of the tenant response.

## Completion

Connector names are described by their tagline. `--version` lists versions
newest first, marks the latest and shortens digests. `--set <TAB>` lists the
schema keys of the selected version, required keys first, each with the first
sentence of its description and its allowed values; `--set KEY=<TAB>` offers
those values. Completion reads only what is typed before the cursor, so type
`--version` or `--channel` before `--set`. Without either, `install` completion
uses the stable release and falls back to the connector's latest version when it
has no stable release. Free-text flags such as `--ledger`, `--name`,
`--poll-interval`, `--query` and `--cursor` never complete file names; only
`--config`, `--env-file` and `--set KEY=@` do.

Help text wraps flag descriptions to the terminal width and shows examples.

## Reading status

`configuration.version` or `configuration.channel` is the requested selector;
`status.appliedVersion` identifies the version applied by the control plane.
Lifecycle is the last recorded verdict (`pending`, `active`, `suspended`,
`completed`, `failed`, or `unknown`), not a live health probe.
`status.ingestion.progress.current` is the recorded ingestion progress.
`status.lastError` is the raw last recorded worker or connector error. It may be
historical after recovery and can be present without ingestion status. Show and
JSON output preserve this text without interpreting JSON-looking content; a successful
read does not fail merely because the recorded error is populated.
Suspension sources identify a tenant request or a Policy by name.

Authentication failures, including insufficient scope, may return `401`
without a JSON body. Check the selected profile and its permissions instead
of assuming every `401` means an expired session.

API error responses use `errorCode` and `errorMessage`; optional `details` is a
string, which may itself contain a JSON object. Commands return generated SDK
errors directly to FCTL’s standard error printer. JSON API errors retain the
schema-defined fields in `connectivity.Error`; undocumented fields and HTTP
metadata are not retained by that type. Responses without a matching JSON schema,
including an empty-body `401`, use `sdkerrors.SDKError` with the status and body.
A missing connector version propagates the API error without a CLI-specific
conversion. Authentication uses Connectivity’s unchanged go-libs middleware.

Required-flag and completion registration failures return a contextual command
setup error before execution, rather than panicking during command construction.
