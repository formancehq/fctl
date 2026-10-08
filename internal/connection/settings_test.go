package connection_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
)

func commandWith(s *connection.Settings, args ...string) *cobra.Command {
	root := &cobra.Command{Use: "test"}
	s.Bind(root)
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root
}

func TestResolvePrecedence(t *testing.T) {
	dir := t.TempDir()
	store := connection.Store{Active: "local", Connections: map[string]connection.Entry{"local": {Options: connection.Options{AuthMode: "none", LedgerURL: "http://saved:9000"}}}}
	if err := connection.Save(dir, store); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FCTL_LEDGER_URL", "http://environment:9000")
	s := &connection.Settings{}
	root := commandWith(s, "--config-dir", dir, "--ledger-url", "http://flag:9000")
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		opts, _, _, _, err := s.Resolve(cmd)
		if err != nil {
			return err
		}
		if opts.LedgerURL != "http://flag:9000" {
			t.Fatalf("resolved %s", opts.LedgerURL)
		}
		if opts.AuthMode != "none" {
			t.Fatal("lost saved auth mode")
		}
		return nil
	}
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestDirectOverrideDoesNotInheritCloudIdentity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entry := connection.NewEntry(connection.Options{AuthMode: "cloud", Issuer: "https://membership.example.com", Organization: "org", Stack: "stack"})
	if err := connection.Save(dir, connection.Store{Active: "cloud", Connections: map[string]connection.Entry{"cloud": entry}}); err != nil {
		t.Fatal(err)
	}
	s := &connection.Settings{}
	root := commandWith(s, "--config-dir", dir, "--auth-mode", "none", "--ledger-url", "http://localhost:9000")
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		opts, _, _, _, err := s.Resolve(cmd)
		if err != nil {
			return err
		}
		if opts.Issuer != "" || opts.Organization != "" || opts.Stack != "" {
			t.Fatal("direct command inherited Cloud identity")
		}
		return connection.Validate(opts)
	}
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestUnauthenticatedGatewayAndDirect(t *testing.T) {
	for _, tc := range []struct{ flag, path string }{{"--stack-url", "/gateway/api/ledger/v3/"}, {"--ledger-url", "/gateway/v3/"}} {
		t.Run(tc.flag, func(t *testing.T) {
			server := httptest.NewServer(anonymousHandler(t, tc.path))
			t.Cleanup(server.Close)
			s := &connection.Settings{}
			root := commandWith(s, "--config-dir", t.TempDir(), tc.flag, server.URL+"/gateway", "--auth-mode", "none")
			root.RunE = func(cmd *cobra.Command, _ []string) error {
				client, err := s.Client(cmd.Context(), cmd, "ledger")
				if err != nil {
					return err
				}
				_, err = client.Do(cmd.Context(), http.MethodGet, "/v3/", nil, nil, nil)
				return err
			}
			if err := root.ExecuteContext(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientCredentialsAndRedirectIsolation(t *testing.T) {
	t.Setenv("FCTL_CLIENT_SECRET", "private-secret")
	stolen := false
	other := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { stolen = true }))
	t.Cleanup(other.Close)
	server := httptest.NewServer(credentialsHandler(t, other.URL))
	t.Cleanup(server.Close)
	s := &connection.Settings{}
	root := commandWith(s, "--config-dir", t.TempDir(), "--ledger-url", server.URL, "--auth-mode", "client-credentials", "--client-id", "service-client", "--token-url", server.URL+"/token")
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := s.Client(cmd.Context(), cmd, "ledger")
		if err != nil {
			return err
		}
		_, err = client.Do(cmd.Context(), http.MethodGet, "/v3/", nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "302") {
			return fmt.Errorf("unexpected redirect result: %w", err)
		}
		return nil
	}
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if stolen {
		t.Fatal("followed redirect with credentials")
	}
}

func TestTokenErrorsHideSecrets(t *testing.T) {
	t.Setenv("FCTL_CLIENT_SECRET", "private-secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error":"invalid_client","error_description":"private-secret"}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	s := &connection.Settings{}
	root := commandWith(s, "--config-dir", t.TempDir(), "--ledger-url", server.URL, "--auth-mode", "client-credentials", "--client-id", "client", "--token-url", server.URL+"/token")
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := s.Client(cmd.Context(), cmd, "ledger")
		if err != nil {
			return err
		}
		_, err = client.Do(cmd.Context(), http.MethodGet, "/v3/", nil, nil, nil)
		return err
	}
	err := root.ExecuteContext(t.Context())
	if err == nil || strings.Contains(err.Error(), "private-secret") {
		t.Fatalf("unsafe error: %v", err)
	}
}

func TestStoreIsPrivateAndRejectsEscapingSymlink(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "config")
	store := connection.Store{Active: "local", Connections: map[string]connection.Entry{"local": {Options: connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9000"}}}}
	if err := connection.Save(dir, store); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode=%v", info.Mode())
	}
	loaded, err := connection.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Active != "local" {
		t.Fatal("store failed roundtrip")
	}
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte(`{"active":"stolen"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "connections.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "connections.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Load(dir); err == nil {
		t.Fatal("followed escaping configuration symlink")
	}
}

func TestValidateConfiguration(t *testing.T) {
	t.Parallel()
	for _, opts := range []connection.Options{
		{},
		{AuthMode: "none", LedgerURL: "http://local", ClientID: "client"},
		{AuthMode: "client-credentials", LedgerURL: "http://local"},
		{AuthMode: "cloud", Issuer: "https://issuer", Organization: "org"},
		{AuthMode: "cloud", Issuer: "https://issuer", Organization: "org", Stack: "stack", LedgerURL: "https://other"},
		{AuthMode: "none", LedgerURL: "https://user:secret@host"}, //nolint:gosec // Invalid synthetic URL is a rejection fixture, not a credential.
	} {
		if connection.Validate(opts) == nil {
			t.Error("accepted invalid configuration")
		}
	}
}

func TestCanceledDirectRequest(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	s := &connection.Settings{}
	root := commandWith(s, "--config-dir", t.TempDir(), "--ledger-url", "http://127.0.0.1:1", "--auth-mode", "none")
	var out bytes.Buffer
	root.SetOut(&out)
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := s.Client(cmd.Context(), cmd, "ledger")
		if err != nil {
			return err
		}
		_, err = client.Do(cmd.Context(), http.MethodGet, "/v3/", nil, nil, nil)
		return err
	}
	if err := root.ExecuteContext(ctx); err == nil {
		t.Fatal("canceled command succeeded")
	}
}

func anonymousHandler(t *testing.T, path string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected authorization")
		}
		if _, err := w.Write([]byte(`{"data":[]}`)); err != nil {
			t.Error(err)
		}
	})
}
func credentialsHandler(t *testing.T, redirect string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			serveCredentialToken(t, w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing bearer")
		}
		http.Redirect(w, r, redirect, http.StatusFound)
	})
}
func serveCredentialToken(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		t.Error(err)
	}
	id, secret, _ := r.BasicAuth()
	if id != "service-client" || secret != "private-secret" {
		t.Error("credentials missing")
	}
	if r.PostForm.Get("grant_type") != "client_credentials" {
		t.Error("wrong grant")
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)); err != nil {
		t.Error(err)
	}
}
