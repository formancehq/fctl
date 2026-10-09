package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func TestProfilesPreserveExistingStoreAndSelectionPrecedence(t *testing.T) {
	dir, existing := existingProfiles(t)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, tc := range []struct {
		name string
		env  string
		args []string
		want connection.Options
	}{
		{"default", "", nil, connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9000"}},
		{"environment", "other", nil, connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9001"}},
		{"long flag", "other", []string{"--profile", "cloud"}, connection.Options{AuthMode: "cloud", Issuer: "https://cloud.example.com"}},
		{"short flag", "other", []string{"-p", "local"}, connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9000"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FCTL_PROFILE", tc.env)
			assertProfileSettings(t, dir, tc.args, tc.want)
			data, err := root.ReadFile("connections.json")
			if err != nil || !bytes.Equal(data, existing) {
				t.Fatalf("profile selection changed the existing store: %v", err)
			}
		})
	}
}

func TestProfilesSwitchPreservesCloudSession(t *testing.T) {
	t.Parallel()
	dir, _ := existingProfiles(t)
	if _, _, err := executeRoot(t, "--config-dir", dir, "profiles", "use", "other"); err != nil {
		t.Fatal(err)
	}
	store, err := connection.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	session := store.Connections["cloud"].Session
	if store.Active != "other" || session == nil || session.MembershipToken == nil || session.MembershipToken.RefreshToken != "fixture-refresh-token" {
		t.Fatal("switching profiles lost the default selection or existing Cloud session")
	}
}

func existingProfiles(t *testing.T) (string, []byte) {
	t.Helper()
	dir := t.TempDir()
	existing := []byte(`{
		"active":"local",
		"connections":{
			"local":{"revision":"local-revision","options":{"authMode":"none","ledgerURL":"http://localhost:9000"}},
			"other":{"revision":"other-revision","options":{"authMode":"none","ledgerURL":"http://localhost:9001"}},
			"cloud":{"revision":"cloud-revision","options":{"authMode":"cloud","issuer":"https://cloud.example.com"},
				"session":{"id_token":"fixture-id-token","membership_token":{"access_token":"fixture-access-token","refresh_token":"fixture-refresh-token","token_type":"Bearer"}}}
		}
	}`)
	if err := os.WriteFile(filepath.Join(dir, "connections.json"), existing, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, existing
}

func assertProfileSettings(t *testing.T, dir string, args []string, want connection.Options) {
	t.Helper()
	args = append([]string{"--config-dir", dir}, args...)
	stdout, stderr, err := executeRoot(t, append(args, "profiles", "show")...)
	if err != nil || stderr != "" {
		t.Fatalf("profile selection failed: %v, stderr=%q", err, stderr)
	}
	var got connection.Options
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
	if strings.Contains(stdout, "fixture-") {
		t.Fatal("profile show exposed saved tokens")
	}
}

func TestProfileLifecycle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, stderr, err := executeRoot(t, append([]string{"--config-dir", dir}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		if stderr != "" {
			t.Fatalf("unexpected stderr: %s", stderr)
		}
		if !json.Valid([]byte(out)) {
			t.Fatalf("invalid JSON: %s", out)
		}
		return out
	}
	run("profiles", "add", "local", "--auth-mode", "none", "--ledger-url", "http://localhost:9000")
	run("profiles", "add", "other", "--auth-mode", "none", "--ledger-url", "http://localhost:9001")
	run("profiles", "use", "other")
	if out := run("profiles", "show"); !strings.Contains(out, "9001") {
		t.Fatal(out)
	}
	if out := run("--profile", "local", "profiles", "show"); !strings.Contains(out, "9000") {
		t.Fatal(out)
	}
	if out := run("profiles", "list"); !strings.Contains(out, `"loggedIn": false`) {
		t.Fatal(out)
	}
	run("profiles", "add", "other", "--replace", "--auth-mode", "none", "--ledger-url", "http://localhost:9002")
	run("profiles", "delete", "other", "--confirm")
	if out := run("profiles", "list"); strings.Contains(out, `"name": "other"`) {
		t.Fatal(out)
	}
	info, err := os.Stat(filepath.Join(dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode: %v", info.Mode())
	}
}

func TestProfileErrorsAndLoginBoundaries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, _, err := executeRoot(t, "--config-dir", dir, "profiles", "add", "local", "--auth-mode", "none", "--ledger-url", "http://localhost:9000"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"profiles", "add", "local", "--auth-mode", "none", "--ledger-url", "http://localhost:9000"},
		{"profiles", "add", "../escape", "--auth-mode", "none", "--ledger-url", "http://localhost:9000"},
		{"profiles", "delete", "local"},
		{"profiles", "delete", "missing", "--confirm"},
		{"profiles", "use", "missing"},
		{"--profile", "missing", "profiles", "show"},
		{"login", "--profile", "local"},
	} {
		if _, _, err := executeRoot(t, append([]string{"--config-dir", dir}, args...)...); err == nil {
			t.Fatalf("accepted invalid command %v", args)
		}
	}
	if _, _, err := executeRoot(t, "--config-dir", dir, "logout"); err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedLedgerAndExternalAuthUseProfile(t *testing.T) {
	t.Parallel()
	server := profileServicesServer(t)
	dir := t.TempDir()
	if _, _, err := executeRoot(t, "--config-dir", dir, "profiles", "add", "local", "--stack-url", server.URL+"/prefix", "--auth-mode", "none"); err != nil {
		t.Fatal(err)
	}
	cacheDistributionAuth(t, dir, pluginmanager.Target{Profile: "local", Endpoint: server.URL + "/prefix/api/auth"})
	for _, args := range [][]string{{"ledger", "list"}, {"auth", "clients", "list"}} {
		out, _, err := executeRoot(t, append([]string{"--config-dir", dir}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, `"data": []`) {
			t.Fatal(out)
		}
	}
}

func profileServicesServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "" {
			t.Error("anonymous connection sent credentials")
		}
		if r.Method == http.MethodGet && r.URL.Path == "/prefix/api/auth/_info" {
			writeCLICloudJSON(t, w, map[string]string{"version": "1.0.0"})
			return
		}
		if r.URL.Path != "/prefix/api/ledger/v3/" && r.URL.Path != "/prefix/api/auth/clients" {
			t.Errorf("unexpected route: %s", r.URL.Path)
		}
		if _, err := w.Write([]byte(`{"data":[]}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}
