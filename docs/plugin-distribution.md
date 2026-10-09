# External Ledger plugin pilot

The Ledger plugin can run as an independent native executable. Its source and
release configuration live in the Ledger repository's `fctl-plugin` Go module.
The public SDK is a separate module at
`github.com/formancehq/fctl/pkg/pluginsdk`. Service plugins depend on this SDK;
they do not depend on fctl's commands, profiles, forms or authentication code.
The embedded Ledger provider imports the same product package.

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

## Public OCI distribution

For each Ledger service version, build six executables for Linux, macOS and
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
| `service` | `ledger` |
| `serviceVersion` | Exact version reported by the Ledger service |
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

New Ledger targets discover native releases from the official catalogue. If
it has no matching release, they retain the embedded provider. A temporary
catalogue transport failure also retains the embedded provider for unprepared
targets; invalid metadata remains an error. An empty catalogue does not query
the service version or trigger target selection.

Set `FCTL_PLUGIN_CATALOGUE` to override the catalogue used for automatic
preparation. Explicit `plugins sync` prefers `--catalogue`, then the environment
variable, then the target's saved catalogue, then the official URL.
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

Each target has a lock identified by profile, organization, stack and stable
connection endpoint. Two stacks in the same profile can use different Ledger
versions. Executables are cached by digest and platform under the private v4
configuration directory's `plugins` folder.

Normal commands retain the locked plugin revision. A changed service version
selects the matching plugin automatically when the target has a catalogue;
fctl prints a preparation notice on stderr. A locally installed binary instead
reports the mismatch and must be replaced. Version checks run before writes.

An explicit `plugins sync` selects the highest plugin revision for the exact
service version. A CLI-only fix increments that revision without requiring a
new Ledger server release. CI can pin both values:

```sh
fctl plugins sync --catalogue ./catalogue.json --profile local \
  --service-version 3.0.0 --revision 1 --no-input
```

Help and completion read the target's cached manifest without network access
or a running plugin process. Prepared commands can use cached executables
without downloading from the registry again. Service operations still need
access to the Ledger service and its version endpoint.

## Validation

`just pc` checks the core and SDK modules. `just tests` runs both race suites.
The CLI tests publish actual plugin executables to an HTTP OCI test registry,
download and execute two service versions, retain and upgrade plugin revisions,
and check offline help and completion. Additional tests cover host OAuth2 and
debug redaction, exact large JSON integers, partial bulk errors, cancellation,
endpoint restrictions, corrupt downloads and concurrent cache installation.

The SDK and product module can also be built and tested independently. Production
OCI publication and updating the official catalogue are separate release
steps; snapshot packaging does not perform either action.
