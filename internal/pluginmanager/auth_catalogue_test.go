package pluginmanager

import (
	"errors"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func authManifest(version string) pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "auth", Service: "auth", Version: version, ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: "auth", Short: "Auth", Subcommands: []pluginsdk.CommandSpec{{Use: "clients", Runnable: true}}}}
}

func TestAuthCatalogueExactVersionAndIndependentLocks(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	ledger := publish(t, m, registry, "3.0.0", 1)
	auth, err := m.Publish(t.Context(), registry.server.URL+"/formance/ledger", Release{Service: "auth", ServiceVersion: "1.0.0", Revision: 1, Platform: CurrentPlatform(), Manifest: authManifest("1.0.0")}, localBinary(t, []byte("auth fixture")))
	if err != nil {
		t.Fatal(err)
	}
	connectivity, err := m.Publish(t.Context(), registry.server.URL+"/formance/ledger", Release{Service: "connectivity", ServiceVersion: "2.0.0", Revision: 1, Platform: CurrentPlatform(), Manifest: func() pluginsdk.Manifest {
		m := authManifest("2.0.0")
		m.Name = "connectivity"
		m.Service = "connectivity"
		m.Root.Use = "connectivity"
		return m
	}()}, localBinary(t, []byte("connectivity fixture")))
	if err != nil {
		t.Fatal(err)
	}
	catalogue := catalogueFile(t, ledger, auth, connectivity)
	for service, version := range map[string]string{"ledger": "3.0.0", "auth": "1.0.0", "connectivity": "2.0.0"} {
		release, err := m.Resolve(t.Context(), catalogue, service, version, 0)
		if err != nil {
			t.Fatal(err)
		}
		lock, err := m.InstallResolved(t.Context(), catalogue, target("shared"), release)
		if err != nil || lock.Service != service {
			t.Fatalf("install %s: %+v %v", service, lock, err)
		}
	}
	if _, err := m.Resolve(t.Context(), catalogue, "auth", "1.0.1", 0); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("Auth version fallback: %v", err)
	}
	registry.server.Close()
	for _, service := range []string{"ledger", "auth", "connectivity"} {
		lock, err := m.Load(target("shared"), service)
		if err != nil || lock.Service != service || lock.Manifest.Service != service {
			t.Fatalf("independent offline lock %s: %+v %v", service, lock, err)
		}
	}
	locks, err := m.List()
	if err != nil || len(locks) != 3 {
		t.Fatalf("multi-service locks: %+v %v", locks, err)
	}
}

func TestAuthCatalogueRejectsVersionAndIdentityMismatchBeforeDownload(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release, err := m.Publish(t.Context(), registry.server.URL+"/formance/ledger", Release{Service: "auth", ServiceVersion: "1.0.0", Revision: 1, Platform: CurrentPlatform(), Manifest: authManifest("1.0.0")}, localBinary(t, []byte("auth fixture")))
	if err != nil {
		t.Fatal(err)
	}
	before := registry.requests.Load()
	for _, test := range []struct {
		name   string
		mutate func(*Release)
	}{
		{"wrong manifest service", func(r *Release) { r.Manifest.Service = "ledger" }},
		{"wrong exact version", func(r *Release) { r.Manifest.Version = "1.0.1" }},
		{"wrong command root", func(r *Release) { r.Manifest.Root.Use = "ledger" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := release
			test.mutate(&invalid)
			catalogue := catalogueFile(t, invalid)
			if _, err := m.Resolve(t.Context(), catalogue, "auth", "1.0.0", 0); err == nil {
				t.Fatal("invalid Auth catalogue accepted")
			}
			if _, err := m.InstallResolved(t.Context(), catalogue, target("reject"), invalid); err == nil {
				t.Fatal("invalid resolved Auth release installed")
			}
			if registry.requests.Load() != before {
				t.Fatal("invalid metadata reached registry")
			}
			if _, err := m.Load(target("reject"), "auth"); !errors.Is(err, ErrNotInstalled) {
				t.Fatalf("invalid metadata wrote lock: %v", err)
			}
		})
	}
}
