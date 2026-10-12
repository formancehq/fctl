package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

type registryFixture struct {
	server        *httptest.Server
	mu            sync.Mutex
	objects       map[string][]byte
	override      map[string][]byte
	length        map[string]int
	requests      atomic.Int32
	manifestReads atomic.Int32
	tokenRequests atomic.Int32
	denyToken     atomic.Bool
}

func newRegistry(t *testing.T) *registryFixture {
	t.Helper()
	registry := &registryFixture{objects: make(map[string][]byte), override: make(map[string][]byte), length: make(map[string]int)}
	registry.server = httptest.NewServer(http.HandlerFunc(registry.serve))
	t.Cleanup(registry.server.Close)
	return registry
}

func (r *registryFixture) serve(writer http.ResponseWriter, request *http.Request) {
	r.requests.Add(1)
	if request.URL.Path == "/token" {
		r.serveToken(writer, request)
		return
	}
	if request.Header.Get("Authorization") != "Bearer public-token" {
		writer.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm=%q,service="test-registry",scope="repository:formance/ledger:pull,push"`, r.server.URL+"/token"))
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch request.Method {
	case http.MethodPost:
		writer.Header().Set("Location", "/v2/formance/ledger/blobs/uploads/fixture?upload-state=keep")
		writer.WriteHeader(http.StatusAccepted)
	case http.MethodPut:
		r.servePut(writer, request)
	case http.MethodGet:
		r.serveGet(writer, request)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *registryFixture) serveToken(writer http.ResponseWriter, request *http.Request) {
	r.tokenRequests.Add(1)
	if r.denyToken.Load() {
		writer.WriteHeader(http.StatusForbidden)
		return
	}
	if request.URL.Query().Get("service") != "test-registry" ||
		!strings.HasPrefix(request.URL.Query().Get("scope"), "repository:formance/ledger:") {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writeFixture(writer, []byte(`{"access_token":"public-token"}`))
}

func (r *registryFixture) servePut(writer http.ResponseWriter, request *http.Request) {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	path := request.URL.Path
	digest := filepath.Base(path)
	if strings.Contains(path, "/blobs/uploads/") {
		digest = request.URL.Query().Get("digest")
		if request.URL.Query().Get("upload-state") != "keep" {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		path = "/v2/formance/ledger/blobs/" + digest
	}
	if digest != "sha256:"+checksum(data) {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	r.objects[path] = data
	writer.Header().Set("Docker-Content-Digest", digest)
	writer.WriteHeader(http.StatusCreated)
}

func (r *registryFixture) serveGet(writer http.ResponseWriter, request *http.Request) {
	data, ok := r.objects[request.URL.Path]
	if replacement, exists := r.override[request.URL.Path]; exists {
		data, ok = replacement, true
	}
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	if strings.Contains(request.URL.Path, "/manifests/") {
		r.manifestReads.Add(1)
		writer.Header().Set("Content-Type", ImageManifestMediaType)
	}
	if length, exists := r.length[request.URL.Path]; exists {
		writer.Header().Set("Content-Length", strconv.Itoa(length))
	}
	writeFixture(writer, data)
}

func writeFixture(writer http.ResponseWriter, data []byte) {
	// Deliberate client cancellation can close the test server's connection.
	if _, err := writer.Write(data); err != nil {
		return
	}
}

func manifest(version string) pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "ledger", Service: "ledger", Version: version, ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: "ledger", Short: "Ledger", Subcommands: []pluginsdk.CommandSpec{{Use: "list", Runnable: true}}}}
}

func manager(t *testing.T) *Manager {
	t.Helper()
	m, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func localBinary(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fctl-plugin-ledger")
	if err := os.WriteFile(path, data, 0o700); err != nil { //nolint:gosec // Fixture deliberately represents a native executable.
		t.Fatal(err)
	}
	return path
}

func publish(t *testing.T, m *Manager, registry *registryFixture, version string, revision int) Release {
	t.Helper()
	data := fmt.Appendf(nil, "native-plugin-%s-%d\n", version, revision)
	release, err := m.Publish(t.Context(), registry.server.URL+"/formance/ledger", Release{
		Service: "ledger", ServiceVersion: version, Revision: revision, Platform: CurrentPlatform(), Manifest: manifest(version),
	}, localBinary(t, data))
	if err != nil {
		t.Fatal(err)
	}
	return release
}

func catalogueFile(t *testing.T, releases ...Release) string {
	t.Helper()
	data, err := json.Marshal(Catalogue{SchemaVersion: SchemaVersion, Releases: releases})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "catalogue.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func target(stack string) Target {
	return Target{Profile: "sandbox", Organization: "example", Stack: stack, Endpoint: "https://app.formance.cloud"}
}

func TestPublishSyncExactVersionsRevisionsAndOfflineTargets(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	v1 := publish(t, m, registry, "3.0.0", 1)
	v2 := publish(t, m, registry, "3.0.0", 2)
	vNext := publish(t, m, registry, "3.0.1", 1)
	catalogue := catalogueFile(t, v1, v2, vNext)
	first, err := m.SyncRevision(t.Context(), catalogue, target("one"), "ledger", "3.0.0", 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Sync(t.Context(), catalogue, target("two"), "ledger", "3.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != 1 || second.Revision != 2 || second.ServiceVersion != "3.0.0" {
		t.Fatalf("incorrect exact selection: first=%+v, second=%+v", first, second)
	}
	if first.ArtifactDigest == second.ArtifactDigest {
		t.Fatal("different revisions unexpectedly share an artifact")
	}
	if first.Catalogue != catalogue {
		t.Fatalf("catalogue was not retained: %q", first.Catalogue)
	}
	if _, err := m.Resolve(t.Context(), catalogue, "ledger", "3.0.2", 0); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("got %v", err)
	}
	if _, err := m.Resolve(t.Context(), catalogue, "ledger", "3.0.0", 3); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("got %v", err)
	}
	assertOfflineTargets(t, m, registry, catalogue, second)
}

func assertOfflineTargets(t *testing.T, m *Manager, registry *registryFixture, catalogue string, second Lock) {
	t.Helper()
	registry.server.Close()
	before := registry.requests.Load()
	loaded, err := m.Load(target("one"), "ledger")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != 1 || len(loaded.Manifest.Root.Subcommands) != 1 {
		t.Fatal("cached help manifest changed")
	}
	if _, err := m.Binary(loaded); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Binary(second); err != nil {
		t.Fatal(err)
	}
	// A new target reuses the already verified digest without a registry call.
	oneOff := Target{Endpoint: "http://localhost:3068"}
	if _, err := m.Sync(t.Context(), catalogue, oneOff, "ledger", "3.0.0"); err != nil {
		t.Fatal(err)
	}
	locks, err := m.List()
	if err != nil || len(locks) != 3 {
		t.Fatalf("list: %v, %d locks", err, len(locks))
	}
	if registry.requests.Load() != before {
		t.Fatal("offline cache attempted network access")
	}
	if registry.tokenRequests.Load() == 0 {
		t.Fatal("bearer token flow was not exercised")
	}
}

func TestInstallFailurePreservesPreviousLock(t *testing.T) {
	for _, failure := range []string{"manifest", "config", "binary", "truncated"} {
		t.Run(failure, func(t *testing.T) { assertInstallFailure(t, failure) })
	}
}

func assertInstallFailure(t *testing.T, failure string) {
	t.Helper()
	m := manager(t)
	registry := newRegistry(t)
	first := publish(t, m, registry, "3.0.0", 1)
	second := publish(t, m, registry, "3.0.1", 1)
	if _, err := m.Install(t.Context(), target("one"), first); err != nil {
		t.Fatal(err)
	}
	registry.mu.Lock()
	path := corruptDownload(registry, second, failure)
	registry.mu.Unlock()
	if _, err := m.Install(t.Context(), target("one"), second); err == nil {
		t.Fatal("corrupt download accepted")
	}
	loaded, err := m.Load(target("one"), "ledger")
	if err != nil || loaded.ArtifactDigest != first.Artifact.Digest {
		t.Fatalf("previous lock changed: %v, %+v", err, loaded)
	}
	if _, err := m.Binary(loaded); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.directory, binaryPath(newLock(target("one"), second)))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed executable published: %v", err)
	}
	registry.mu.Lock()
	delete(registry.override, path)
	delete(registry.length, path)
	registry.mu.Unlock()
	if _, err := m.Install(t.Context(), target("one"), second); err != nil {
		t.Fatalf("retry did not recover: %v", err)
	}
}

func corruptDownload(registry *registryFixture, release Release, failure string) string {
	path := "/v2/formance/ledger/blobs/sha256:" + release.SHA256
	switch failure {
	case "manifest":
		path = "/v2/formance/ledger/manifests/" + release.Artifact.Digest
	case "config":
		path = "/v2/formance/ledger/blobs/sha256:" + checksum([]byte("{}"))
	}
	original := registry.objects[path]
	if failure == "truncated" {
		registry.override[path] = original[:2]
		registry.length[path] = len(original)
	} else {
		corrupt := append([]byte(nil), original...)
		corrupt[0] ^= 1
		registry.override[path] = corrupt
	}
	return path
}

func TestConcurrentInstallSharesDownloadAndCancelsWait(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release := publish(t, m, registry, "3.0.0", 1)
	var wait sync.WaitGroup
	failures := make(chan error, 12)
	for i := range 12 {
		wait.Go(func() {
			_, err := m.Install(t.Context(), target(strconv.Itoa(i)), release)
			failures <- err
		})
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if registry.manifestReads.Load() != 1 {
		t.Fatalf("downloaded manifest %d times", registry.manifestReads.Load())
	}
	locks, err := m.List()
	if err != nil || len(locks) != 12 {
		t.Fatalf("list: %v, %d locks", err, len(locks))
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := m.Install(ctx, target("cancelled"), release); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestCancellationWhileWaitingForInstallationLock(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release := publish(t, m, registry, "3.0.0", 1)
	held := make(chan struct{})
	unlock := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- m.withCacheLock(t.Context(), newLock(target("holder"), release), func(*os.Root) error {
			close(held)
			<-unlock
			return nil
		})
	}()
	<-held
	before := registry.requests.Load()
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	_, err := m.Install(ctx, target("waiting"), release)
	close(unlock)
	if cleanupErr := <-finished; cleanupErr != nil {
		t.Fatal(cleanupErr)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if registry.requests.Load() != before {
		t.Fatal("a cancelled lock waiter downloaded an executable")
	}
	if _, err := m.Load(target("waiting"), "ledger"); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("cancelled waiter published a lock: %v", err)
	}
}

func TestInstallLocalCopiesAndVerifiesWithEndpointIdentity(t *testing.T) {
	m := manager(t)
	original := localBinary(t, []byte("original-executable"))
	one := target("one")
	two := one
	two.Endpoint = "http://localhost:3068/api/ledger"
	first, err := m.InstallLocal(t.Context(), original, one, manifest("3.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.InstallLocal(t.Context(), original, two, manifest("3.0.0")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}
	path, err := m.Binary(first)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Local || first.Catalogue != "" {
		t.Fatalf("incorrect local lock %+v", first)
	}
	locks, err := m.List()
	if err != nil || len(locks) != 2 {
		t.Fatalf("endpoint identity collided: %v, %d", err, len(locks))
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0o700); err != nil { //nolint:gosec // Keep the fixture executable to specifically test checksum validation.
		t.Fatal(err)
	}
	if _, err := m.Binary(first); err == nil {
		t.Fatal("corrupted cache executable accepted")
	}
	if _, err := m.Load(one, "ledger"); err != nil {
		t.Fatalf("metadata-only help should remain available: %v", err)
	}
}

func TestHTTPDownloadDeadlineAndAnonymousAuthDenied(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release := publish(t, m, registry, "3.0.0", 1)
	registry.denyToken.Store(true)
	if _, err := m.Install(t.Context(), target("one"), release); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("got %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	if _, err := m.Resolve(ctx, server.URL, "ledger", "3.0.0", 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}
