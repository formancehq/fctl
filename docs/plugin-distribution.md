# External service plugin distribution

Ledger, Auth and Connectivity plugins can run as independent native executables. Their product
modules own their commands and executable entry points:

| Service | Product module | Executable |
| --- | --- | --- |
| Ledger | `github.com/formancehq/ledger/misc/fctl-plugin` | `fctl-plugin-ledger` |
| Auth | `github.com/formancehq/auth/misc/fctl-plugin` | `fctl-plugin-auth` |
| Connectivity | `github.com/formancehq/connectivity/misc/fctl-plugin` | `fctl-plugin-connectivity` |

Auth product sources remain in the Auth repository. Public executable distribution
is a separate publication step and does not require publishing private sources.
The public SDK is a separate module at
`github.com/formancehq/fctl/pkg/pluginsdk`. Service plugins depend on this SDK;
they do not depend on fctl's commands, profiles, forms or authentication code.
Embedded providers and external executables use the same product commands.

## Try a local executable

Build from the Ledger checkout using the exact service version on your target:

```sh
nix develop --command just build-fctl-plugin 3.0.0-beta.10 1
./build/fctl-plugin-ledger --version
./build/fctl-plugin-ledger --manifest
```

From fctl, prepare the selected target and then use ordinary Ledger commands:

```sh
fctl plugins install --binary /path/to/ledger/build/fctl-plugin-ledger \
  --organization ORGANIZATION_ID --stack STACK_ID
fctl ledger list --organization ORGANIZATION_ID --stack STACK_ID
fctl ledger create --organization ORGANIZATION_ID --stack STACK_ID
fctl plugins show --organization ORGANIZATION_ID --stack STACK_ID
fctl plugins list
```

Installation discovers the version through the selected Ledger's `/_info` and
rejects a binary whose manifest targets another version. Authentication and
debug traces still belong to fctl. Local endpoints support `none` and
`client-credentials`, and Cloud stacks use the existing Cloud profile.
Terminal forms and tables work with both providers. No plugin-specific flags
are added to ordinary Ledger commands.

## Auth installation

Build the Auth executable from its product module, injecting the exact deployed
Auth service version. The executable supports `--manifest` and `--version` for
inspection. From the Auth checkout:

```sh
cd misc/fctl-plugin
go build -ldflags "-X main.serviceVersion=1.0.0 -X main.revision=1" \
  -o /tmp/fctl-plugin-auth ./cmd/fctl-plugin-auth
/tmp/fctl-plugin-auth --manifest
/tmp/fctl-plugin-auth --version
```

Use the built executable through the generic host:

```sh
fctl plugins install --service auth --binary /tmp/fctl-plugin-auth \
  --auth-mode none --auth-url http://localhost:8080
fctl auth clients list --auth-mode none --auth-url http://localhost:8080
fctl plugins show --service auth --auth-mode none --auth-url http://localhost:8080
fctl plugins sync --service auth --catalogue ./registry.yaml --profile local
```

`plugins install`, `plugins sync` and `plugins show` accept `--service ledger`
or `--service auth` or `--service connectivity`; the default remains `ledger`. Unknown services are rejected
before installation. The selected service must match the manifest service, name
and command root. The manifest version must equal the exact service version.
Without `--service-version`, installation and sync discover that version through
the selected service's `/_info`. An explicit version supports preparation without
contacting the service; execution still verifies the deployed version.

## Public OCI distribution

For each product service version, build six executables for Linux, macOS and
Windows on amd64 and arm64. Publish them as public OCI artifacts. No container
engine is required on the user's machine. Keeping service sources private does
not require keeping their CLI binaries private; package visibility must be
configured explicitly in the registry.

The product's publisher generates catalogue entries from the actual manifest,
platform builds, checksums and published OCI digests. Publish every artifact
before advertising its mapping. The catalogue has `schemaVersion: 1` and a
`releases` array. Each entry contains:

| Field | Meaning |
| --- | --- |
| `service` | `ledger` or `auth` |
| `serviceVersion` | Exact version reported by the selected service; identical to `manifest.version` |
| `revision` | Positive plugin revision for that service version |
| `platform` | Operating system and architecture |
| `artifact` | Registry origin, repository and immutable `sha256:` manifest digest |
| `sha256` | SHA-256 of the raw executable layer |
| `manifest` | Complete SDK command and form manifest, including protocol version |

The pilot accepts a trusted YAML or JSON catalogue file or HTTPS URL. HTTP is
allowed only for loopback test registries and catalogues. The default catalogue
is the public `registry.yaml` in `formancehq/fctl-plugin-registry`:

```text
https://raw.githubusercontent.com/formancehq/fctl-plugin-registry/main/registry.yaml
```

```sh
fctl plugins sync --profile local
fctl plugins sync --catalogue ./registry.yaml --profile local
fctl ledger list --profile local
```

New Ledger and Auth targets discover native releases from the official catalogue. If
it has no matching release, they retain the embedded provider. A temporary
catalogue transport failure also retains the embedded provider for unprepared
targets; invalid metadata remains an error. An empty catalogue does not query
the service version or trigger target selection.

Set `FCTL_PLUGIN_CATALOGUE` to override the catalogue used for automatic
preparation. Explicit `plugins sync` prefers `--catalogue`, then the environment
variable, then the selected service's saved catalogue, then the official URL.
The core reads `/_info`, selects the exact version and current platform, checks
protocol compatibility, downloads the digest-addressed artifact and verifies
the manifest, config and executable checksums. A missing exact release fails
for explicit sync, custom catalogues and already prepared external targets.
The stack supplies its service version; it never
supplies an executable URL.

Checksums verify bytes against the trusted catalogue. They do not independently
authenticate its publisher. Protect catalogue publication and registry write
access. Only explicitly trusted local binaries should be installed.

## Locks and updates

Locks are identified by service plus profile, organization, stack and stable
connection endpoint. Auth, Ledger and Connectivity keep independent locks for the same target.
Two stacks in the same profile can use different service versions. Executables
are cached by digest and platform under the private v4
configuration directory's `plugins` folder.

Normal commands retain the locked plugin revision. A changed service version
selects the matching plugin automatically when the target has a catalogue;
fctl prints a preparation notice on stderr. A locally installed binary instead
reports the mismatch and must be replaced. Version checks run before writes.

An explicit `plugins sync` selects the highest plugin revision for the exact
service version. A CLI-only fix increments that revision without requiring a
new product server release. CI can pin both values:

```sh
fctl plugins sync --catalogue ./catalogue.json --profile local \
  --service-version 3.0.0 --revision 1 --no-input
```

Help and completion read the target's cached manifest without network access
or a running plugin process. Prepared commands can use cached executables
without downloading from the registry again. Service operations still need
access to the selected service and its version endpoint.

## Validation

`just pc` checks the core and SDK modules. `just tests` runs both race suites.
The CLI tests publish actual Ledger executables and public-SDK Auth fixture
executables to HTTP OCI test registries on localhost,
download and execute two service versions, retain and upgrade plugin revisions,
and check offline help and completion. Auth tests also reject mismatched catalogue
identity and exact versions before downloads or lock writes, and verify independent
Auth and Ledger locks. Product integration must separately validate the Auth executable
built from its product module. Additional tests cover host OAuth2 and
debug redaction, exact large JSON integers, partial bulk errors, cancellation,
endpoint restrictions, corrupt downloads and concurrent cache installation.

The SDK and product module can also be built and tested independently. Production
OCI publication and updating the official catalogue are separate release
steps; snapshot packaging does not perform either action.

Connectivity has no embedded provider or product Go dependency. Use the exact API version, independent of Core and connector versions. Cached help/completion remain available offline. Version checks now run on every execution, including interactive discovery; there is an extra info request per operation.
