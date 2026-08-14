# Connectivity commands

`fctl connectivity` manages the published Connector catalogue and the
ConnectorInstances installed in the selected stack.

## Command tree

```text
fctl connectivity connectors list [--filter KEY=VALUE] [--query JSON]
fctl connectivity connectors show <connector>
fctl connectivity connectors facets [--filter KEY=VALUE] [--query JSON]

fctl connectivity connectorinstances list [--connector NAME] [--filter KEY=VALUE]
fctl connectivity connectorinstances show <instance>
fctl connectivity connectorinstances install <connector> --ledger <ledger>
fctl connectivity connectorinstances configure <instance>
fctl connectivity connectorinstances suspend <instance> --confirm
fctl connectivity connectorinstances unsuspend <instance> --confirm
fctl connectivity connectorinstances uninstall <instance>
```

Short aliases are available for the common list and catalogue commands:
`connectors ls`, `connectors l`, `connectors facets` (`facet`, `f`), and
`connectorinstances ls` (`l`). `install` also accepts `create` and `in`;
`configure` accepts `config`, `update`, and `c`.

## Suspension and resumption

`connectorinstances suspend` and `connectorinstances unsuspend` request the
desired ingestion state through the Connectivity API. Both commands require
confirmation, either interactively or with `--confirm`:

```bash
fctl connectivity connectorinstances suspend stripe-prod --confirm
fctl connectivity connectorinstances unsuspend stripe-prod --confirm
```

A successful command means the request was accepted; reconciliation may still
be in progress. Use `connectorinstances show <instance>` to compare the desired
`Suspend` and `Replicas` values with the observed phase, state, and
`Suspended by` provenance. Suspension can also remain effective because the
desired replica count is zero or a matching Policy contributes a suspension
source.

## Filtering and pagination

List and facets commands accept repeatable `--filter` expressions:

```bash
fctl connectivity connectors list --filter catalog=ee --filter tags~provider:%
fctl connectivity connectorinstances list --filter channel=beta
```

`KEY=VALUE` is an exact match, `KEY!=VALUE` excludes a present matching key,
and `KEY~PATTERN` uses SQL `%` and `_` wildcards. Use `--query '<JSON>'` for
the full API query dialect; it cannot be combined with `--filter`. The CLI
also accepts `--page-size` and the opaque `--cursor` from a previous response.
Shell completion obtains the currently supported keys, values, and channels
from the API.

## Versions and channels

An install without `--version` or `--channel` resolves its configuration schema
from the **stable** channel, matching the API's persisted default. `--version`
pins an exact release. `--channel stable|rc|beta|alpha` tracks that channel;
when both are present the explicit version wins.

During `configure`, `--channel` removes a prior pin unless `--version` is also
provided. The CLI resolves the channel across every catalogue page using the
same rules as Connectivity: it never downgrades and stays within the currently
resolved major version. `connectors show` and schema-driven configuration use
the API's `latest` alias only where that is the documented catalogue view.

## Configuration inputs

`install` and `configure` accept three configuration forms, in precedence
order: `--config` (YAML or JSON), repeatable `--env-file`, then repeatable
`--set KEY=VALUE`. Values may be inline, `@path`/`@-`,
`secret://name/key`, or `configmap://name/key`. Structured config supports
separate `env` and `files` sections.

The CLI derives known keys and definitely-required fields from the Connector
Version JSON Schema, including bounded local `$ref`, `allOf`, `anyOf`, and
`oneOf` composition. Open or pattern-based keys, and constraints that cannot
be proven from CLI input, are sent to Connectivity for authoritative
validation.
