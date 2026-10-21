# Cloud commands and hosted helpers

`cloud` is the control-plane command root. Its embedded plugin uses Membership
for identity, organizations, regions and stack metadata. Auth and Ledger remain
separate module roots. This page describes the current implementation; live
validation evidence belongs in [validation.md](validation.md).

| Command group | Purpose | Access boundary |
| --- | --- | --- |
| `cloud me` | Identity and personal invitations | Membership identity |
| `cloud organizations` | Organizations, users, invitations, policies, OAuth clients, authentication providers and enabled applications | Membership organization grant |
| `cloud regions` | Regions and available stack versions | Membership organization grant |
| `cloud stack` | Stack lifecycle, modules, users and history | Membership organization grant |
| `cloud apps` | Experimental apps, manifests, deployments and variables | Separate Deploy application grant |

Cloud metadata commands require a logged-in Cloud profile. Organization and
stack selection use explicit flags, saved defaults, or a unique available ID.
They do not need a stack gateway connection to read Membership metadata.
See [profiles and authentication](profiles.md) for authentication and defaults.

```sh
fctl login --profile cloud
fctl cloud me info
fctl cloud organizations list
fctl cloud regions list --organization ORGANIZATION_ID
fctl cloud stack list --organization ORGANIZATION_ID
fctl cloud stack show STACK_ID --organization ORGANIZATION_ID -o json
fctl cloud stack modules list STACK_ID --organization ORGANIZATION_ID
```

## Organization updates

`cloud organizations update` preserves an existing positive `defaultPolicyID`
when omitted. If the current policy is absent, `null` or `0`, explicitly choose
a positive ID using `--default-policy-id` or JSON `defaultPolicyID`. Membership
defaults an unset policy to Guest (ID `4`, with read access), so the CLI requires
that choice instead of sending an unset value. Explicit `null` or `0` cannot
clear the policy and is rejected before PUT.

## Stack lifecycle

`cloud stack` provides `create`, `list`, `show`, `update`, `delete`, `disable`,
`enable`, `restore`, `upgrade`, `history`, `info` and `version`. Nested `modules`
commands list, enable or disable modules; nested `users` commands list, link or
unlink users. Consult each command's `--help` for its arguments, JSON body and
confirmation requirements.

Stack creation defaults to version `v4.0`. Version selection must match the
region's version catalog; it does not silently fall back to another version.
Creation, restoration and upgrades wait for Membership to report `READY` and
the requested version. `--no-wait` returns after acceptance; `--wait-timeout`
defaults to `10m`. These checks do not prove gateway or module runtime health.

`list --all` includes deleted and disabled stacks. History supports page size,
cursor, action, user and data filters; a cursor cannot be combined with the
other history filters. `info` and `version` report Membership metadata, rather
than querying each running stack component.

## Hosted helpers

The host attaches these utilities to the Cloud command tree:

| Command | Behavior |
| --- | --- |
| `cloud stack proxy` | Authenticated gateway proxy on `127.0.0.1`, port `55001` by default; `--port 0` selects an available port |
| `cloud stack mcp serve` | Bridges stdin/stdout MCP messages to the gateway's `/api/mcp`; only `--transport stdio` is supported |
| `cloud generate-personal-token` (alias `gpt`) | Prints the selected stack bearer token as JSON, with color disabled |

Helpers use the host's stack connection resolver. The proxy removes incoming
authentication headers and cookies before the authenticated transport sends the
request. `--allowed-origins` configures browser access. Its listening address is
written to stderr. Proxy and MCP operation follow command cancellation; MCP
stdout is reserved for protocol messages. Personal-token output contains a
credential and must be handled accordingly.

## Experimental Apps

`cloud apps` is distinct from `cloud organizations applications`, which manages
applications enabled for an organization. Every Apps command requires
`--experimental=true`, including reads. The default-false gate is checked before
connection resolution or body-file access. Destructive commands also require
`--confirm=true`.

```sh
fctl cloud apps list --experimental --organization ORGANIZATION_ID
fctl cloud apps deployments list --experimental --organization ORGANIZATION_ID
```

The host resolves `cloud-apps` through a separate application grant. The default
`--deploy-app-alias` is `deploy`; the grant uses `app://ALIAS` and granted
`apps:Read`/`apps:Write` scopes. The backend endpoint comes from the signed
application token's audience. The Membership root credential is not sent to
Deploy, and the stack Auth token exchange is not used for Apps.

Apps supports apps, manifests and versions, deployments, and variables.
Manifest uploads accept the declared `--path` flag or a JSON body containing
`yaml`. Downloads support `--out`. Deployment creation waits by default;
`--wait=false` disables waiting. The default wait timeout is `30m`, with a
maximum of `24h`. These are implementation contracts, not a claim that a live
deployment has been verified.

## Migration from v3

| Previous command | Current command |
| --- | --- |
| `fctl stack list` | `fctl cloud stack list` |
| `fctl stack show STACK_ID` | `fctl cloud stack show STACK_ID` |
| `fctl stack proxy` | `fctl cloud stack proxy` |
| `fctl stack mcp serve` | `fctl cloud stack mcp serve` |

V3 profiles are not migrated automatically. Create a v4 profile and log in;
use explicit organization/stack flags or saved defaults. Auth and Ledger still
use their own module roots. For automation, select `-o json` explicitly; see
[output.md](output.md) for terminal defaults and diagnostics.
