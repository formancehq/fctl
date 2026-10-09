package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func loaderManifest(service string) pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: service, Service: service, Version: "1.0.0", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: service, Short: "Cached external " + service, Subcommands: []pluginsdk.CommandSpec{{Use: "probe", Short: "External probe", Runnable: true}}}}
}

func loaderSettings(t *testing.T) (*cobra.Command, *connection.Settings) {
	t.Helper()
	root := &cobra.Command{Use: "fctl"}
	settings := &connection.Settings{}
	settings.Bind(root)
	settings.Directory = t.TempDir()
	loaderFlag(t, root, "auth-mode", "none")
	return root, settings
}

func loaderFlag(t *testing.T, root *cobra.Command, name, value string) {
	t.Helper()
	if err := root.PersistentFlags().Set(name, value); err != nil {
		t.Fatal(err)
	}
}

func TestServiceLoaderTargets(t *testing.T) {
	for _, service := range []string{"auth", "ledger"} {
		t.Run(service, func(t *testing.T) { testServiceLoaderTarget(t, service) })
	}
	root, settings := loaderSettings(t)
	if _, err := pluginServiceTarget(settings, root, "unknown"); err == nil {
		t.Fatal("unknown service accepted")
	}
	if _, err := cachedServiceLock(settings, root, nil, "unknown"); err == nil {
		t.Fatal("unknown cached service accepted")
	}
	if _, _, err := pluginServiceSyncTarget(t.Context(), settings, root, "unknown", "1.0.0"); err == nil {
		t.Fatal("unknown sync service accepted")
	}
	if _, err := prepareServicePlugin(t.Context(), root, settings, nil, "unknown"); err == nil {
		t.Fatal("unknown provider accepted")
	}
}

func testServiceLoaderTarget(t *testing.T, service string) {
	t.Helper()
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "stack-url", "https://stack.example/")
	target, err := pluginServiceTarget(settings, root, service)
	if err != nil || target.Endpoint != "https://stack.example/api/"+service {
		t.Fatalf("stack target: %+v %v", target, err)
	}
	loaderFlag(t, root, "auth-url", "https://auth.example")
	loaderFlag(t, root, "ledger-url", "https://ledger.example")
	target, err = pluginServiceTarget(settings, root, service)
	if err != nil || target.Endpoint != "https://"+service+".example" {
		t.Fatalf("direct target: %+v %v", target, err)
	}
	loaderFlag(t, root, "auth-mode", "cloud")
	target, err = pluginServiceTarget(settings, root, service)
	if err != nil || target.Endpoint != cloud.DefaultIssuer {
		t.Fatalf("cloud default: %+v %v", target, err)
	}
	loaderFlag(t, root, "issuer", "https://issuer.example")
	target, err = pluginServiceTarget(settings, root, service)
	if err != nil || target.Endpoint != settings.Options.Issuer {
		t.Fatalf("cloud identity: %+v %v", target, err)
	}
}

func TestServiceLoaderManifestBoundary(t *testing.T) {
	root, _ := loaderSettings(t)
	for _, service := range []string{"auth", "ledger"} {
		manifest := loaderManifest(service)
		if err := validateServiceManifest(t.Context(), root, service, manifest); err != nil {
			t.Fatal(err)
		}
		for _, change := range []func(*pluginsdk.Manifest){
			func(m *pluginsdk.Manifest) { m.Name = "cloud" },
			func(m *pluginsdk.Manifest) { m.Service = "cloud" },
			func(m *pluginsdk.Manifest) { m.Root.Use = "cloud" },
			func(m *pluginsdk.Manifest) { m.Root.Service = "cloud" },
			func(m *pluginsdk.Manifest) { m.Root.Subcommands[0].Service = "cloud-apps" },
		} {
			invalid := loaderManifest(service)
			change(&invalid)
			if err := validateServiceManifest(t.Context(), root, service, invalid); err == nil {
				t.Fatalf("accepted foreign identity/boundary: %+v", invalid)
			}
		}
	}
	if err := validateServiceManifest(t.Context(), root, "cloud", loaderManifest("cloud")); err == nil {
		t.Fatal("unsupported manifest accepted")
	}
}

func installLoaderFixture(t *testing.T, manager *pluginmanager.Manager, target pluginmanager.Target, service string) pluginmanager.Lock {
	t.Helper()
	path := filepath.Join(t.TempDir(), "not-an-executable")
	if err := os.WriteFile(path, []byte("cached help must never start this file"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := manager.InstallLocal(t.Context(), path, target, loaderManifest(service))
	if err != nil {
		t.Fatal(err)
	}
	return lock
}

//nolint:gocognit // One offline lifecycle verifies two cached providers and ambiguity isolation.
func TestServiceLoaderOfflineRootsAndCacheIsolation(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "https://invalid.example/catalogue")
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "auth-mode", "cloud")
	loaderFlag(t, root, "issuer", "https://issuer.example")
	manager, err := pluginManager(settings, root)
	if err != nil {
		t.Fatal(err)
	}
	target := pluginmanager.Target{Profile: "", Organization: "org", Stack: "stack", Endpoint: settings.Options.Issuer}
	for _, service := range []string{"auth", "ledger"} {
		installLoaderFixture(t, manager, target, service)
	}
	for _, service := range []string{"auth", "ledger"} {
		lock, err := cachedServiceLock(settings, root, manager, service)
		if err != nil || lock.Service != service {
			t.Fatalf("cached %s: %+v %v", service, lock, err)
		}
		for _, suffix := range [][]string{{service, "--help"}, {"__complete", service, ""}} {
			args := append([]string{"--config-dir", settings.Directory, "--auth-mode", "cloud", "--issuer", settings.Options.Issuer}, suffix...)
			command := NewRootCommandWithArgs(t.Context(), args)
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)
			command.SetArgs(args)
			if err := command.ExecuteContext(t.Context()); err != nil {
				t.Fatalf("offline %v: %v", suffix, err)
			}
			if !strings.Contains(output.String(), "probe") {
				t.Fatalf("cached metadata not registered: %s", output.String())
			}
			count := 0
			for _, child := range command.Commands() {
				if child.Name() == service {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("%s roots: %d", service, count)
			}
		}
	}
	installLoaderFixture(t, manager, pluginmanager.Target{Profile: "", Organization: "org", Stack: "other", Endpoint: settings.Options.Issuer}, "auth")
	if _, err := cachedServiceLock(settings, root, manager, "auth"); !errors.Is(err, pluginmanager.ErrNotInstalled) {
		t.Fatalf("ambiguous Auth metadata: %v", err)
	}
	if lock, err := cachedServiceLock(settings, root, manager, "ledger"); err != nil || lock.Service != "ledger" {
		t.Fatalf("Auth ambiguity affected Ledger: %+v %v", lock, err)
	}
}

func TestAuthLoaderOfficialFallbackAndOffline(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	for _, test := range []struct {
		name, data string
		status     int
		embedded   bool
	}{
		{"empty", "schemaVersion: 1\nreleases: []\n", http.StatusOK, true},
		{"unavailable", "", http.StatusServiceUnavailable, true},
		{"invalid", "schemaVersion: 1\nreleases: invalid\n", http.StatusOK, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			prep := discoveryPreparation(t, discoveryTransport(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != pluginmanager.DefaultCatalogue {
					t.Fatalf("unexpected service request: %s", request.URL)
				}
				return &http.Response{StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.data))}, nil
			}))
			prep.service = serviceDescriptor{name: "auth", title: "Auth"}
			loaderFlag(t, prep.root, "auth-url", "http://127.0.0.1:1")
			_, err := prep.resolve(t.Context(), pluginBootstrap{commands: []string{"auth", "probe"}})
			if err == nil || errors.Is(err, pluginmanager.ErrNotInstalled) != test.embedded {
				t.Fatalf("Auth fallback: %v", err)
			}
		})
	}
	prep := discoveryPreparation(t, discoveryTransport(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("offline Auth accessed network: %s", request.URL)
		return nil, context.Canceled
	}))
	prep.service = serviceDescriptor{name: "auth", title: "Auth"}
	loaderFlag(t, prep.root, "auth-url", "http://127.0.0.1:1")
	for _, plan := range []pluginBootstrap{{commands: []string{"auth"}, help: true}, {commands: []string{"__complete", "auth"}}, {commands: []string{"ledger", "list"}}, {}} {
		if _, err := prep.resolve(t.Context(), plan); !errors.Is(err, pluginmanager.ErrNotInstalled) {
			t.Fatalf("offline Auth: %v", err)
		}
	}
}

func TestAuthLoaderExactVersionAndStrictTargets(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	base := http.DefaultTransport
	http.DefaultTransport = discoveryTransport(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "http://127.0.0.1:1/_info" {
			t.Fatalf("wrong Auth version endpoint: %s", request.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"version":"v1.0.1"}`))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = base })
	prep := discoveryPreparation(t, discoveryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	prep.service = serviceDescriptor{name: "auth", title: "Auth"}
	loaderFlag(t, prep.root, "auth-mode", "none")
	loaderFlag(t, prep.root, "auth-url", "http://127.0.0.1:1")
	target, version, err := pluginServiceSyncTarget(t.Context(), prep.settings, prep.root, "auth", "")
	if err != nil || version != "1.0.1" {
		t.Fatalf("exact Auth discovery: %+v %s %v", target, version, err)
	}
	empty := &pluginmanager.Catalogue{SchemaVersion: pluginmanager.SchemaVersion}
	if _, err := prep.matchVersion(t.Context(), pluginmanager.DefaultCatalogue, empty); !errors.Is(err, pluginmanager.ErrNotInstalled) {
		t.Fatalf("official unprepared missing release: %v", err)
	}
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "https://custom.example/catalogue")
	if _, err := prep.resolve(t.Context(), pluginBootstrap{commands: []string{"auth", "probe"}}); !errors.Is(err, pluginmanager.ErrCatalogueUnavailable) {
		t.Fatalf("custom unavailable target fell back: %v", err)
	}
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	installLoaderFixture(t, prep.manager, target, "auth")
	if _, err := prep.matchVersion(t.Context(), pluginmanager.DefaultCatalogue, empty); !errors.Is(err, pluginmanager.ErrNoRelease) {
		t.Fatalf("prepared target accepted missing exact release: %v", err)
	}
	if _, err := prep.matchVersion(t.Context(), pluginmanager.DefaultCatalogue, nil); !errors.Is(err, pluginmanager.ErrCatalogueUnavailable) {
		t.Fatalf("prepared unavailable target fell back: %v", err)
	}
}

func TestLedgerLoaderCompatibility(t *testing.T) {
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "ledger-url", "https://ledger.example")
	target, err := ledgerTarget(settings, root)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := pluginManager(settings, root)
	if err != nil {
		t.Fatal(err)
	}
	installLoaderFixture(t, manager, target, "ledger")
	if _, err := cachedLedgerLock(settings, root, manager); err != nil {
		t.Fatal(err)
	}
	if _, err := soleLedgerLock(manager, target); err != nil {
		t.Fatal(err)
	}
	if _, version, err := pluginSyncTarget(t.Context(), settings, root, "v1.0.0"); err != nil || version != "1.0.0" {
		t.Fatalf("legacy sync: %s %v", version, err)
	}
	if err := ledgerManifestIdentity(loaderManifest("ledger")); err != nil {
		t.Fatal(err)
	}
	if err := validateLedgerManifest(t.Context(), root, loaderManifest("ledger")); err != nil {
		t.Fatal(err)
	}
	if factory, err := prepareLedgerPlugin(t.Context(), root, settings, nil); err != nil || factory != nil {
		t.Fatalf("embedded-only legacy root: %v", err)
	}
	client, err := api.New("https://ledger.example", &http.Client{Transport: discoveryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"version":"v1.0.0"}`))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if version, err := ledgerServiceVersion(t.Context(), client); err != nil || version != "1.0.0" {
		t.Fatalf("legacy version: %s %v", version, err)
	}
}
