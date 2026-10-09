# External service plugin distribution

Auth and Ledger run only as external native executables. Their generic host
loader and plugin management already exist; the local Ledger embedded fallback
has been removed. Auth official discovery is available. No official Ledger
release is currently advertised in the catalogue: a future Ledger product
release must publish executables and promote their entries. Trusted local
Ledger builds can be installed in the meantime.

Connectivity commands are absent after removal of the local adapter. An
independently distributed Connectivity plugin must be integrated later;
`plugins install/sync/show` currently support only Auth and Ledger. Host
connection settings are preserved. These changes do not complete the migration.

The Auth and Ledger product modules own their commands and executable entry points:

| Service | Product module | Executable |
| --- | --- | --- |
| Ledger | `github.com/formancehq/ledger/misc/fctl-plugin` | `fctl-plugin-ledger` |
| Auth | `github.com/formancehq/auth/misc/fctl-plugin` | `fctl-plugin-auth` |

Auth product sources remain in the Auth repository. Public executable distribution
is a separate publication step and does not require publishing private sources.
The public SDK is a separate module at
`github.com/formancehq/fctl/pkg/pluginsdk`. Service plugins depend on this SDK;
they do not depend on fctl's commands, profiles, forms or authentication code.
Auth and Ledger commands are not compiled into fctl through local service
adapters. Their command manifests come from separately installed executables;
Cloud remains embedded.

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
Terminal forms and tables use the external plugin manifest. No plugin-specific
flags
are added to ordinary Ledger commands.

## Auth installation

Build the Auth executable from its independent product module with Go 1.26 or
the product's Nix environment. You need access to an Auth checkout containing
the merged plugin changes. Use the exact deployed Auth version from `/_info`,
without its leading `v`, and a positive plugin revision. The version below is
an example: replace it with your target's version. From the Auth checkout root:

```sh
export AUTH_SERVICE_VERSION=2.5.0-beta.1
export PLUGIN_REVISION=1
cd misc/fctl-plugin
GOWORK=off go build \
  -ldflags "-X main.serviceVersion=$AUTH_SERVICE_VERSION -X main.revision=$PLUGIN_REVISION" \
  -o /tmp/fctl-plugin-auth ./cmd/fctl-plugin-auth
/tmp/fctl-plugin-auth --manifest
/tmp/fctl-plugin-auth --version
```

Alternatively, from the Auth checkout root:

```sh
nix develop --impure --command just build-fctl-plugin 2.5.0-beta.1 1
./build/fctl-plugin-auth --manifest
./build/fctl-plugin-auth --version
```

Install the module-built executable for a local Auth target. This example
assumes Auth is running at `http://localhost:8080` and permits anonymous access:

```sh
fctl plugins install --service auth --binary /tmp/fctl-plugin-auth \
  --auth-mode none --auth-url http://localhost:8080
fctl auth clients list --auth-mode none --auth-url http://localhost:8080
fctl plugins show --service auth \
  --auth-mode none --auth-url http://localhost:8080
fctl auth --help --auth-mode none --auth-url http://localhost:8080
fctl completion zsh --auth-mode none --auth-url http://localhost:8080
```

For the Nix build, use `--binary ./build/fctl-plugin-auth` instead. Installation
reads the binary's manifest and discovers the deployed version; it rejects a
mismatch. To prepare without contacting Auth, add
`--service-version "$AUTH_SERVICE_VERSION"` to the install command. The supplied
version must match the binary; execution still verifies the deployed version.
The last two commands use the installed target's complete cached metadata and
remain usable with Auth offline. Without that cache, `fctl auth --help` shows
only a placeholder guiding `plugins sync --service auth` or
`plugins install --service auth`; it never contacts a catalogue or Auth.

Auth official discovery and explicit sync are available for matching published
versions; the catalogue advertises Auth `2.5.2`, revision `1`. Sync it explicitly
for a target running that exact version. These commands force the official
catalogue,
even if an environment override or saved custom catalogue exists:

```sh
fctl plugins sync --service auth \
  --catalogue https://raw.githubusercontent.com/formancehq/fctl-plugin-registry/main/registry.yaml \
  --auth-mode none --auth-url http://localhost:8080
fctl plugins sync --service auth \
  --catalogue https://raw.githubusercontent.com/formancehq/fctl-plugin-registry/main/registry.yaml \
  --organization ORGANIZATION_ID --stack STACK_ID
```

These examples require a matching published Auth version on the target.
See [Auth release availability](#auth-release-availability) for the recorded
publication evidence. Local installation works independently of publication.
For a custom test catalogue, pass `--catalogue ./registry.yaml` with the same
Auth target flags.

`plugins install`, `plugins sync` and `plugins show` accept `--service ledger`
or `--service auth`; the default remains `ledger`. Unknown services are rejected
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

The default Ledger sync below requires a future official Ledger release.
Today, use a trusted matching custom catalogue or the local installation
procedure above; a missing official entry cannot prepare commands.

```sh
fctl plugins sync --profile local
fctl plugins sync --catalogue ./registry.yaml --profile local
fctl ledger list --profile local
```

Before Auth execution on a fresh target, fctl discovers the official catalogue,
resolves the target's exact service version and platform, and installs the
matching executable. An unavailable official catalogue, an empty catalogue or
no exact matching Auth release fails explicitly before command execution. Auth
never falls back to an embedded provider. A prepared target uses its installed
executable and cached metadata; ordinary execution still checks its version.

Ledger has no embedded provider. The existing external loader and management
commands can use a trusted installed executable or a matching custom catalogue.
The official catalogue currently has no Ledger release, so official sync cannot
prepare Ledger commands. Install a trusted exact-version binary until the Ledger
product publishes and promotes a release. Missing releases or unavailable
metadata do not restore an embedded command tree.

Set `FCTL_PLUGIN_CATALOGUE` to override the catalogue used for automatic
preparation. Explicit `plugins sync` prefers `--catalogue`, then the environment
variable, then the selected service's saved catalogue, then the official URL.
The core reads `/_info`, selects the exact version and current platform, checks
protocol compatibility, downloads the digest-addressed artifact and verifies
the manifest, config and executable checksums. An exact matching release is
required for discovery or sync. A trusted local
installation must also match the target version; there is no embedded fallback.
The stack supplies its service version; it never
supplies an executable URL.

Checksums verify bytes against the trusted catalogue. They do not independently
authenticate its publisher. Protect catalogue publication and registry write
access. Only explicitly trusted local binaries should be installed.

## Locks and updates

Locks are identified by service plus profile, organization, stack and stable
connection endpoint. Auth and Ledger keep independent locks for the same target.
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

Help and completion read the selected target's cached manifest, including its
full command tree and declarative forms, without network access or a running
plugin process. A fresh Auth or Ledger target exposes only sync/install guidance; help
and completion do not trigger automatic discovery. Prepared commands can use
cached executables without downloading from the registry again. Service operations still need
access to the selected service and its version endpoint.

## Validation

`just pc` checks the core and SDK modules. `just tests` runs both race suites.
The CLI tests publish synthetic public-SDK Ledger and Auth fixture executables
to HTTP OCI test registries on localhost,
download and execute two service versions, retain and upgrade plugin revisions,
and check offline help and completion. Auth tests also reject mismatched catalogue
identity and exact versions before downloads or lock writes, and verify independent
Auth and Ledger locks. These fixtures validate the host transport and rendering,
including explicit partial-result errors; they do not validate Ledger business
behavior. Opt-in product tests separately exercise the independently published
Auth executable. Additional tests cover host OAuth2 and
debug redaction, exact large JSON integers, partial bulk errors, cancellation,
endpoint restrictions, corrupt downloads and concurrent cache installation.

The SDK and product module can also be built and tested independently. Production
OCI publication and updating the official catalogue are separate release
steps; snapshot packaging does not perform either action.

## Auth release availability

Auth PR #162 is merged and the `v2.5.2` tagged release has published all six
Auth executables at revision 1. Anonymous GHCR downloads and executable
checksums have been verified. The generated entries were merged through
[registry PR #2](https://github.com/formancehq/fctl-plugin-registry/pull/2),
so the default catalogue now advertises Auth `2.5.2`, revision `1`.

For future releases, the tag must include the plugin module and release tooling
on the tagged branch.

Official availability requires separate release steps:

1. The Auth tagged-release workflow builds the six executables and publishes
   digest-addressed OCI artifacts to `ghcr.io/formancehq/fctl-plugin-auth`.
2. The package administrator enables public access, and maintainers verify
   anonymous downloads and executable checksums. A successful upload or a
   GitHub release asset alone does not prove public download availability.
3. Maintainers promote the generated catalogue entries into
   `formancehq/fctl-plugin-registry/registry.yaml`. Auth's release workflow does
   not update that official catalogue.

Snapshot packaging performs neither GHCR publication nor official catalogue
promotion. Auth `2.5.2` revision `1` has passed anonymous download checks
for all six platforms and fresh-cache discovery through the official catalogue.
The execution check used the published binary with a local API fixture; it did
not upgrade or mutate a deployed Auth service. Targets running other exact
versions need a matching release or a trusted local executable.
