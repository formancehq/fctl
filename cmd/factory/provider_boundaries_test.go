package factory

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	legacy "github.com/formancehq/fctl/misc/fctl-plugin"
	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
	"github.com/formancehq/fctl/v4/internal/pluginselection"
)

func TestProviderBoundaryFirstLegacyAliasInvocation(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	t.Setenv("FCTL_PROFILE", "")
	f := newLegacySelectionFixture(t)
	directory := filepath.Join(f.directory, "plugins")
	target := pluginselection.Target{Endpoint: f.server.URL}
	if _, err := pluginselection.Load(directory, target, "ledger"); !errors.Is(err, pluginselection.ErrNotSelected) {
		t.Fatalf("expected a fresh target: %v", err)
	}
	provider, historical, err := prepareProvider(t.Context(), f.root, f.settings, []string{"l", "list"}, "ledger")
	if err != nil || provider == nil || !historical {
		t.Fatalf("first l list did not select legacy: provider=%v historical=%v err=%v", provider != nil, historical, err)
	}
	registry := &plugin.Registry{}
	if err := registry.Register(t.Context(), provider(nil), provider); err != nil {
		t.Fatal(err)
	}
	if err := plugin.NewCommandWithRequest(registry, serviceRequestResolver(f.root, f.settings, map[string]bool{"ledger": historical})).AddTo(f.root); err != nil {
		t.Fatal(err)
	}
	cmd, remaining, err := f.root.Find([]string{"l", "list"})
	if err != nil || len(remaining) != 0 || cmd.CommandPath() != "fctl ledger list" {
		t.Fatalf("legacy alias was not registered: command=%s remaining=%v err=%v", cmd.CommandPath(), remaining, err)
	}
	selection, err := pluginselection.Load(directory, target, "ledger")
	if err != nil || selection.Provider != "legacy" || selection.ServiceVersion != "2.4.15" || selection.StackVersion != "" || selection.PluginVersion != legacy.Version {
		t.Fatalf("standalone alias selection: %+v err=%v", selection, err)
	}
	if f.callCount() != 1 {
		t.Fatalf("first alias should inspect the service once, got %d requests", f.callCount())
	}
}

func TestProviderBoundaryResolverFreezesDirectIdentity(t *testing.T) {
	t.Setenv("FCTL_PROFILE", "")
	for _, historical := range []bool{true, false} {
		t.Run(map[bool]string{true: "legacy", false: "modern"}[historical], func(t *testing.T) {
			testProviderBoundaryDirectIdentity(t, historical)
		})
	}
}

func testProviderBoundaryDirectIdentity(t *testing.T, historical bool) {
	t.Helper()
	f := newLegacySelectionFixture(t)
	roots := map[string]bool{"ledger": historical}
	resolver := serviceRequestResolver(f.root, f.settings, roots)
	// Model a later tree builder and another CLI replacing the selection.
	roots["ledger"] = !historical
	selected := pluginselection.Selection{Target: pluginselection.Target{Endpoint: f.server.URL}, Service: "ledger", ServiceVersion: "2.4.15", Provider: "modern"}
	if !historical {
		selected.Provider, selected.PluginVersion = "legacy", legacy.Version
	}
	if err := pluginselection.Save(filepath.Join(f.directory, "plugins"), selected); err != nil {
		t.Fatal(err)
	}
	client, err := resolver(t.Context(), "ledger", pluginsdk.ExecuteRequest{CommandPath: []string{"ledger", "list"}})
	if err != nil {
		t.Fatal(err)
	}
	metadata := client.Context()
	if (metadata["provider"] == "legacy") != historical {
		t.Fatalf("mutable cache/map changed the registered provider: historical=%v metadata=%v", historical, metadata)
	}
	if historical && metadata["plugin-version"] != legacy.Version {
		t.Fatalf("wrong historical plugin identity: %v", metadata)
	}
	if f.callCount() != 0 {
		t.Fatal("resolving a direct client should not inspect a remote stack")
	}
}

func TestProviderBoundaryLegacyCloudV4GuardSurvivesCacheReplacement(t *testing.T) {
	t.Setenv("FCTL_PROFILE", "")
	f := newProviderBoundaryCloudFixture(t)
	selected := pluginselection.Selection{Target: pluginselection.Target{Profile: "work", Organization: "org", Stack: "stack", Endpoint: f.server.URL + "/api"}, Service: "auth", ServiceVersion: "2.5.1", StackVersion: "3.2.0", Provider: "legacy", PluginVersion: legacy.Version}
	directory := filepath.Join(f.directory, "plugins")
	if err := pluginselection.Save(directory, selected); err != nil {
		t.Fatal(err)
	}
	roots := map[string]bool{"auth": true, "legacy": true}
	resolver := serviceRequestResolver(f.root, f.settings, roots)
	delete(roots, "auth")
	roots["legacy"] = false
	selected.Provider, selected.PluginVersion, selected.StackVersion = "modern", "", "v4.0.0-beta.1"
	if err := pluginselection.Save(directory, selected); err != nil {
		t.Fatal(err)
	}
	for _, path := range [][]string{{"auth", "clients", "create"}, {"legacy", "auth", "clients", "create"}} {
		client, err := resolver(t.Context(), "auth", pluginsdk.ExecuteRequest{CommandPath: path})
		if client != nil || err == nil || !strings.Contains(err.Error(), "legacy plugin "+legacy.Version) || !strings.Contains(err.Error(), "v4.0.0-beta.1") {
			t.Fatalf("%v bypassed the Cloud v4 guard: client=%v err=%v", path, client != nil, err)
		}
	}
	if f.stackReads.Load() != 2 || f.operations.Load() != 0 {
		t.Fatalf("guard should read Membership before any operation: stack reads=%d operations=%d", f.stackReads.Load(), f.operations.Load())
	}
}

func TestProviderBoundaryCorruptModernLockDoesNotFallBack(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	t.Setenv("FCTL_PROFILE", "")
	for _, data := range []string{`{"schemaVersion":`, `{"schemaVersion":999}`} {
		t.Run(data, func(t *testing.T) {
			f := newLegacySelectionFixture(t)
			installProviderBoundaryLock(t, f)
			lockDir := filepath.Join(f.directory, "plugins", "targets")
			entries, err := os.ReadDir(lockDir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("expected one installed lock: entries=%v err=%v", entries, err)
			}
			if err := os.WriteFile(filepath.Join(lockDir, entries[0].Name()), []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			provider, historical, err := prepareProvider(t.Context(), f.root, f.settings, []string{"ledger", "list"}, "ledger")
			if err == nil || errors.Is(err, pluginmanager.ErrNotInstalled) || provider != nil || historical {
				t.Fatalf("corrupt modern lock fell back: provider=%v historical=%v err=%v", provider != nil, historical, err)
			}
			if _, err := pluginselection.Load(filepath.Join(f.directory, "plugins"), pluginselection.Target{Endpoint: f.server.URL}, "ledger"); !errors.Is(err, pluginselection.ErrNotSelected) {
				t.Fatalf("failed preparation published a selection: %v", err)
			}
		})
	}
}

func TestProviderBoundaryModernVersionMismatchLeavesOfflineHelpUnprepared(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	t.Setenv("FCTL_PROFILE", "")
	f := newLegacySelectionFixture(t)
	installProviderBoundaryLock(t, f)
	selected := pluginselection.Selection{Target: pluginselection.Target{Endpoint: f.server.URL}, Service: "ledger", ServiceVersion: "3.0.0-beta.10", Provider: "modern"}
	if err := pluginselection.Save(filepath.Join(f.directory, "plugins"), selected); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"ledger", "--help"}, {"ledger"}, {"__complete", "ledger", ""}} {
		provider, historical, err := prepareProvider(t.Context(), f.root, f.settings, args, "ledger")
		if provider != nil || historical || err == nil || !strings.Contains(err.Error(), "ledger 3.0.0-beta.10 was observed but its plugin is not prepared") {
			t.Fatalf("%v exposed a stale modern manifest: provider=%v historical=%v err=%v", args, provider != nil, historical, err)
		}
	}
	lock, err := pluginhost.Show(f.settings, f.root, "ledger")
	if err != nil || lock.ServiceVersion != "2.4.15" {
		t.Fatalf("offline help changed the old lock: %+v err=%v", lock, err)
	}
	if f.callCount() != 0 {
		t.Fatalf("offline help accessed the network %d times", f.callCount())
	}
}

func installProviderBoundaryLock(t *testing.T, f *legacySelectionFixture) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata-only-plugin")
	if err := os.WriteFile(path, []byte("offline metadata must not execute this file"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := pluginmanager.New(filepath.Join(f.directory, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest := pluginsdk.Manifest{Name: "ledger", Service: "ledger", Version: "2.4.15", ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{Use: "ledger", Subcommands: []pluginsdk.CommandSpec{{Use: "probe", Runnable: true}}}}
	if _, err := manager.InstallLocal(t.Context(), path, pluginmanager.Target{Endpoint: f.server.URL}, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := pluginhost.Show(f.settings, f.root, "ledger"); err != nil {
		t.Fatalf("fixture must start with a valid modern lock: %v", err)
	}
}

type providerBoundaryCloudFixture struct {
	server     *httptest.Server
	directory  string
	root       *cobra.Command
	settings   *connection.Settings
	stackReads atomic.Int32
	operations atomic.Int32
	orgToken   string
}

func newProviderBoundaryCloudFixture(t *testing.T) *providerBoundaryCloudFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &providerBoundaryCloudFixture{directory: t.TempDir(), root: &cobra.Command{Use: "fctl"}, settings: &connection.Settings{}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, key, w, r) }))
	t.Cleanup(f.server.Close)
	entry := f.entry(t, key)
	if err := connection.Save(f.directory, connection.Store{Active: "work", Connections: map[string]connection.Entry{"work": entry}}); err != nil {
		t.Fatal(err)
	}
	f.root.SetContext(t.Context())
	f.settings.Bind(f.root)
	for name, value := range map[string]string{"config-dir": f.directory, "profile": "work", "auth-mode": "cloud", "issuer": f.server.URL + "/api", "client-id": "fctl", "organization": "org", "stack": "stack", "no-browser": "true"} {
		if err := f.root.PersistentFlags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *providerBoundaryCloudFixture) entry(t *testing.T, key *rsa.PrivateKey) connection.Entry {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "key"))
	if err != nil {
		t.Fatal(err)
	}
	issuer := f.server.URL + "/api"
	id := providerBoundaryClaims(t, signer, issuer, map[string]any{"org": []any{map[string]any{"id": "org", "scopes": []string{"organization:Read", "organization:ListStacks"}, "stacks": []any{map[string]any{"id": "stack", "uri": f.server.URL + "/stack", "scopes": []string{"stack:Read", "stack:Write"}}}}}})
	f.orgToken = providerBoundaryClaims(t, signer, issuer, map[string]any{"organization_id": "org", "scope": "organization:Read organization:ListStacks"})
	options := cloud.Options{Issuer: issuer, ClientID: "fctl"}
	orgOptions := options
	orgOptions.Organization = "org"
	stackOptions := orgOptions
	stackOptions.Stack = "stack"
	entry := connection.NewEntry(connection.Options{AuthMode: "cloud", Issuer: issuer, ClientID: "fctl"})
	entry.Session = &cloud.Session{Options: options, IDToken: id, MembershipToken: providerBoundaryToken("root-access"),
		Organizations: map[string]*cloud.Session{"org": {Options: orgOptions, IDToken: id, MembershipToken: providerBoundaryToken(f.orgToken)}},
		Targets:       map[string]*cloud.Session{"org/stack": {Options: stackOptions, IDToken: id, MembershipToken: providerBoundaryToken("stack-membership"), StackURL: f.server.URL + "/stack", StackToken: providerBoundaryToken("stack-access")}}}
	return entry
}

func providerBoundaryClaims(t *testing.T, signer jose.Signer, issuer string, values map[string]any) string {
	t.Helper()
	values["iss"], values["aud"], values["sub"], values["exp"] = issuer, "fctl", "user", time.Now().Add(time.Hour).Unix()
	raw, err := jwt.Signed(signer).Claims(values).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func providerBoundaryToken(access string) *oauth2.Token {
	return &oauth2.Token{AccessToken: access, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
}

func (f *providerBoundaryCloudFixture) serve(t *testing.T, key *rsa.PrivateKey, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodGet {
		f.operations.Add(1)
		t.Errorf("guard made an unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var data any
	switch r.URL.Path {
	case "/api/.well-known/openid-configuration":
		data = map[string]any{"issuer": f.server.URL + "/api", "authorization_endpoint": f.server.URL + "/api/authorize", "device_authorization_endpoint": f.server.URL + "/api/device", "token_endpoint": f.server.URL + "/api/token", "jwks_uri": f.server.URL + "/api/jwks", "id_token_signing_alg_values_supported": []string{"RS256"}}
	case "/api/jwks":
		data = jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "key", Algorithm: "RS256", Use: "sig"}}}
	case "/api/organizations/org/stacks/stack":
		f.stackReads.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+f.orgToken {
			t.Error("stack version did not use the organization Membership grant")
		}
		data = map[string]any{"data": map[string]string{"id": "stack", "organizationId": "org", "version": "v4.0.0-beta.1"}}
	default:
		f.operations.Add(1)
		t.Errorf("guard accessed an unexpected route: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		t.Error(err)
	}
}
