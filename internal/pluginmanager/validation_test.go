package pluginmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMalformedCatalogueAndPlatformAreRejected(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	valid := publish(t, m, registry, "3.0.0", 1)
	for _, test := range []struct {
		name   string
		mutate func(*Release)
	}{
		{"repository traversal", func(r *Release) { r.Artifact.Repository = "../ledger" }},
		{"digest traversal", func(r *Release) { r.Artifact.Digest = "sha256:../../outside" }},
		{"registry credentials", func(r *Release) { r.Artifact.Registry = "https://user:secret@registry.example" }},
		{"registry insecure", func(r *Release) { r.Artifact.Registry = "http://registry.example" }},
		{"registry query", func(r *Release) { r.Artifact.Registry += "?token=secret" }},
		{"registry path", func(r *Release) { r.Artifact.Registry += "/repo" }},
		{"invalid hash", func(r *Release) { r.SHA256 = "bad" }},
		{"service traversal", func(r *Release) { r.Service = "../ledger" }},
		{"platform traversal", func(r *Release) { r.Platform.OS = "../outside" }},
		{"service mismatch", func(r *Release) { r.Manifest.Service = "auth" }},
		{"empty root", func(r *Release) { r.Manifest.Root.Use = "" }},
		{"unsupported protocol", func(r *Release) { r.Manifest.ProtocolVersion++ }},
	} {
		t.Run(test.name, func(t *testing.T) {
			release := valid
			test.mutate(&release)
			if _, err := m.Resolve(t.Context(), catalogueFile(t, release), "ledger", "3.0.0", 0); err == nil {
				t.Fatal("unsafe catalogue entry accepted")
			}
		})
	}
	other := valid
	other.Platform = Platform{OS: "unsupported", Arch: "unsupported"}
	if _, err := m.Resolve(t.Context(), catalogueFile(t, other), "ledger", "3.0.0", 0); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("got %v", err)
	}
	if _, err := m.Install(t.Context(), target("one"), other); err == nil {
		t.Fatal("wrong-platform install accepted")
	}
	if _, err := m.Resolve(t.Context(), catalogueFile(t, valid, valid), "ledger", "3.0.0", 0); err == nil {
		t.Fatal("ambiguous duplicate release accepted")
	}
}

func TestUnsafeTargetsAndForgedLocksAreRejected(t *testing.T) {
	m := manager(t)
	for _, endpoint := range []string{"", "https://user:secret@example.org", "https://example.org?token=secret", "https://example.org#secret", "file:///tmp/ledger"} {
		if _, err := m.Load(Target{Endpoint: endpoint}, "ledger"); err == nil || errors.Is(err, ErrNotInstalled) {
			t.Fatalf("invalid endpoint accepted: %q, %v", endpoint, err)
		}
	}
	if _, err := m.Load(Target{Profile: "evil\x00profile", Endpoint: "http://localhost:3068"}, "ledger"); err == nil {
		t.Fatal("NUL identity accepted")
	}
	if _, err := m.Load(target("one"), "../ledger"); err == nil {
		t.Fatal("unsafe service accepted")
	}
	if _, err := m.Load(target("missing"), "ledger"); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("got %v", err)
	}
	lock, err := m.InstallLocal(t.Context(), localBinary(t, []byte("executable")), target("one"), manifest("3.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	lock.Platform.OS = "../../outside"
	if _, err := m.Binary(lock); err == nil {
		t.Fatal("forged cache path accepted")
	}
}

func TestRootContainsCachePathsAndLeavesNoTemporaryFiles(t *testing.T) {
	for _, directory := range []string{"targets", "artifacts", "guards"} {
		t.Run(directory, func(t *testing.T) {
			m := manager(t)
			outside := t.TempDir()
			if err := os.Symlink(outside, filepath.Join(m.directory, directory)); err != nil {
				t.Fatal(err)
			}
			if _, err := m.InstallLocal(t.Context(), localBinary(t, []byte("executable")), target("one"), manifest("3.0.0")); err == nil {
				t.Fatal("cache symlink escaped root")
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatalf("outside cache was written: %v, %v", entries, err)
			}
		})
	}
}

func TestCatalogueHTTPAndBounds(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release := publish(t, m, registry, "3.0.0", 1)
	data, err := json.Marshal(Catalogue{SchemaVersion: SchemaVersion, Releases: []Release{release}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/catalogue":
			writeFixture(writer, data)
		case "/invalid":
			writeFixture(writer, []byte(`{"schemaVersion":999}`))
		case "/malformed":
			writeFixture(writer, []byte("{"))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	if _, err := m.Sync(t.Context(), server.URL+"/catalogue", target("one"), "ledger", "3.0.0"); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{server.URL + "/invalid", server.URL + "/malformed", server.URL + "/missing", "file:///tmp/catalogue", "http://example.org/catalogue"} {
		if _, err := m.Resolve(t.Context(), source, "ledger", "3.0.0", 0); err == nil {
			t.Fatalf("invalid catalogue accepted: %s", source)
		}
	}
	if _, err := readBounded(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("oversized response accepted")
	}
}

func TestRegistryRedirectDropsBearerToken(t *testing.T) {
	var gotAuthorization string
	blob := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		writeFixture(writer, []byte("binary"))
	}))
	defer blob.Close()
	registry := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, blob.URL, http.StatusTemporaryRedirect)
	}))
	defer registry.Close()
	m := manager(t)
	session := registrySession{client: m.client, registry: registry.URL, repository: "formance/ledger", actions: "pull", token: "private-registry-token"}
	if _, err := session.bytes(t.Context(), registry.URL, 32); err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "" {
		t.Fatal("registry bearer token leaked to redirected host")
	}
}

func TestOCIFormatBoundsAndChecksumAgreement(t *testing.T) {
	base := imageManifest{SchemaVersion: 2, MediaType: ImageManifestMediaType, ArtifactType: ArtifactMediaType,
		Config: descriptor{Digest: "sha256:" + checksum([]byte("{}")), Size: 2},
		Layers: []descriptor{{MediaType: ExecutableMediaType, Digest: "sha256:" + checksum([]byte("binary")), Size: 6}}}
	for _, test := range []struct {
		name   string
		change func(*imageManifest)
	}{
		{"archive", func(m *imageManifest) { m.Layers[0].MediaType = "application/gzip" }},
		{"too large", func(m *imageManifest) { m.Layers[0].Size = maxBinaryBytes + 1 }},
		{"empty binary", func(m *imageManifest) { m.Layers[0].Size = 0 }},
		{"wrong checksum", func(m *imageManifest) { m.Layers[0].Digest = "sha256:" + checksum(nil) }},
		{"config digest", func(m *imageManifest) { m.Config.Digest = "not-a-digest" }},
		{"config size", func(m *imageManifest) { m.Config.Size = maxManifestBytes + 1 }},
		{"image index", func(m *imageManifest) { m.MediaType = "application/vnd.oci.image.index.v1+json" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := base
			copy.Layers = append([]descriptor(nil), base.Layers...)
			test.change(&copy)
			if err := validateImage(copy, checksum([]byte("binary"))); err == nil {
				t.Fatal("invalid OCI format accepted")
			}
		})
	}
}

func TestLocalSourcesMustBeRegularAndNonEmpty(t *testing.T) {
	m := manager(t)
	if _, err := m.InstallLocal(t.Context(), t.TempDir(), target("one"), manifest("3.0.0")); err == nil {
		t.Fatal("directory accepted as executable")
	}
	if _, err := m.InstallLocal(t.Context(), localBinary(t, nil), target("one"), manifest("3.0.0")); err == nil {
		t.Fatal("empty executable accepted")
	}
	if _, err := m.Resolve(t.Context(), t.TempDir(), "ledger", "3.0.0", 0); err == nil {
		t.Fatal("directory accepted as catalogue")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := hashReader(ctx, bytes.NewReader([]byte("binary")), 16); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if _, _, err := hashReader(t.Context(), bytes.NewReader([]byte("binary")), 2); err == nil {
		t.Fatal("unbounded local executable read")
	}
}

func TestBearerChallengeRejectsUnsupportedOrMalformedAuthentication(t *testing.T) {
	for _, challenge := range []string{"", `Basic realm="registry"`, `Bearer realm="https://example.org/token"`, `Bearer realm="unterminated`} {
		if _, _, err := bearerChallenge(challenge); err == nil {
			t.Fatalf("invalid challenge accepted: %s", challenge)
		}
	}
	realm, service, err := bearerChallenge(`Bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:formance/ledger:pull,push"`)
	if err != nil || realm != "https://ghcr.io/token" || service != "ghcr.io" {
		t.Fatalf("GHCR challenge rejected: %q, %q, %v", realm, service, err)
	}
}

func TestInstallResolvedDoesNotRefetchAndOfflineHelpDoesNotNeedBinary(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release := publish(t, m, registry, "3.0.0", 1)
	catalogue := catalogueFile(t, release)
	resolved, err := m.Resolve(t.Context(), catalogue, "ledger", "3.0.0", 0)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a changed catalogue after the host validates the manifest.
	if err := os.WriteFile(catalogue, []byte("invalid replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := m.InstallResolved(t.Context(), catalogue, target("one"), resolved)
	if err != nil {
		t.Fatal(err)
	}
	if lock.Catalogue != catalogue || lock.ArtifactDigest != release.Artifact.Digest {
		t.Fatalf("installed a different resolution: %+v", lock)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := m.BinaryContext(ctx, lock); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled checksum verification returned %v", err)
	}
	binary, err := m.Binary(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(binary); err != nil {
		t.Fatal(err)
	}
	registry.server.Close()
	loaded, err := m.Load(target("one"), "ledger")
	if err != nil || loaded.Manifest.Name != "ledger" {
		t.Fatalf("offline manifest unavailable: %v", err)
	}
	if _, err := m.Binary(loaded); err == nil {
		t.Fatal("execution allowed with a missing binary")
	}
}

func TestInstallResolvedRejectsUntrustedCatalogueSource(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release := publish(t, m, registry, "3.0.0", 1)
	before := registry.requests.Load()
	for _, source := range []string{"file:///tmp/catalogue.json", "http://insecure.example/catalogue", "https://user:secret@example.org/catalogue", "https://example.org/catalogue#fragment"} {
		if _, err := m.InstallResolved(t.Context(), source, target("one"), release); err == nil {
			t.Fatalf("unsafe resolved catalogue source accepted: %q", source)
		}
	}
	if registry.requests.Load() != before {
		t.Fatal("unsafe source caused an artifact download")
	}
}
