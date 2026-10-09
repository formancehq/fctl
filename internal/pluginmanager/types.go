// Package pluginmanager resolves exact service versions from a configured public
// catalogue and installs native executables from digest-addressed OCI artifacts.
// It does not start plugins or access the host's service authentication.
//
// A catalogue has schemaVersion 1 and a releases array. Each release identifies
// service, serviceVersion, revision, platform {os, arch}, artifact {registry,
// repository, digest}, sha256, and the complete pluginsdk manifest. The registry
// is an HTTPS origin (HTTP is allowed for loopback development registries).
// Artifacts use an OCI image manifest, an empty JSON config and exactly one raw
// executable layer; archives and image indexes are deliberately unsupported.
//
// New(cacheDirectory, client) creates the manager. Sync(ctx, catalogue, target,
// service, exactVersion) selects the highest revision for that exact version and
// the current platform, installs it, and atomically writes the target's lock.
// Load and List read only local metadata; Binary verifies the cached executable.
// InstallLocal copies an explicitly selected binary with a manifest obtained by
// the caller. The caller must verify the running binary's manifest against the
// locked manifest before executing commands, for both local and OCI installs.
package pluginmanager

import (
	"context"
	"errors"
	"net/http"
	"runtime"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

const (
	SchemaVersion                = 1
	ArtifactMediaType            = "application/vnd.formance.fctl.plugin.v1"
	ExecutableMediaType          = "application/vnd.formance.fctl.plugin.executable.v1"
	ImageManifestMediaType       = "application/vnd.oci.image.manifest.v1+json"
	EmptyConfigMediaType         = "application/vnd.oci.empty.v1+json"
	maxCatalogueBytes      int64 = 8 << 20
	maxManifestBytes       int64 = 4 << 20
	maxBinaryBytes         int64 = 128 << 20
)

var (
	ErrNotInstalled = errors.New("plugin is not installed for this target")
	ErrNoRelease    = errors.New("no plugin release for the exact service version and platform")
)

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

func CurrentPlatform() Platform { return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH} }

func (p Platform) String() string { return p.OS + "/" + p.Arch }

type Artifact struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Digest     string `json:"digest"`
}

type Release struct {
	Service        string             `json:"service"`
	ServiceVersion string             `json:"serviceVersion"`
	Revision       int                `json:"revision"`
	Platform       Platform           `json:"platform"`
	Artifact       Artifact           `json:"artifact"`
	SHA256         string             `json:"sha256"`
	Manifest       pluginsdk.Manifest `json:"manifest"`
}

type Catalogue struct {
	SchemaVersion int       `json:"schemaVersion"`
	Releases      []Release `json:"releases"`
}

// Target is a stable connection identity, not a transient API gateway URL.
// Profile may be empty for one-off connections. Endpoint is a local service URL
// or a Cloud issuer; it never contains credentials, query values or a fragment.
type Target struct {
	Profile      string `json:"profile"`
	Organization string `json:"organization,omitzero"`
	Stack        string `json:"stack,omitzero"`
	Endpoint     string `json:"endpoint"`
}

type Lock struct {
	SchemaVersion  int                `json:"schemaVersion"`
	Target         Target             `json:"target"`
	Service        string             `json:"service"`
	ServiceVersion string             `json:"serviceVersion"`
	Revision       int                `json:"revision"`
	ArtifactDigest string             `json:"artifactDigest"`
	Platform       Platform           `json:"platform"`
	SHA256         string             `json:"sha256"`
	Manifest       pluginsdk.Manifest `json:"manifest"`
	Catalogue      string             `json:"catalogue,omitzero"`
	Artifact       Artifact           `json:"artifact,omitzero"`
	Local          bool               `json:"local,omitzero"`
	InstalledAt    time.Time          `json:"installedAt"`
}

type Manager struct {
	directory string
	client    *http.Client
	platform  Platform
}

// Resolve reads a catalogue file or URL, without executing a plugin. A zero
// revision selects the highest revision for the exact serviceVersion only.
func (m *Manager) Resolve(ctx context.Context, catalogue, service, serviceVersion string, revision int) (Release, error) {
	return m.resolve(ctx, catalogue, service, serviceVersion, revision)
}

// Install installs a previously resolved release. Sync should usually be used
// instead so that the source catalogue is persisted for later automatic sync.
func (m *Manager) Install(ctx context.Context, target Target, release Release) (Lock, error) {
	return m.install(ctx, "", target, release)
}

// InstallResolved persists the already inspected release without re-fetching
// its catalogue. This lets the host validate manifest collisions first.
func (m *Manager) InstallResolved(ctx context.Context, catalogue string, target Target, release Release) (Lock, error) {
	return m.install(ctx, catalogue, target, release)
}
