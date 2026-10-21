package connection_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
)

//nolint:gocognit // The matrix verifies HTTP paths and token acquisition for each direct/gateway authentication boundary.
func TestConnectivityDirectAndGatewayAuthentication(t *testing.T) {
	t.Setenv("FCTL_CLIENT_SECRET", "fixture-secret")
	for _, mode := range []string{"none", "client-credentials"} {
		for _, flag := range []string{"--connectivity-url", "--stack-url"} {
			t.Run(mode+flag, func(t *testing.T) {
				path := "/prefix/connectorinstances"
				if flag == "--stack-url" {
					path = "/prefix/api/connectivity/connectorinstances"
				}
				tokens, reads := 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/token" {
						tokens++
						r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
						if err := r.ParseForm(); err != nil {
							t.Error(err)
						}
						if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("scope") != "connectivity:read" {
							t.Error("wrong credential grant")
						}
						if _, err := io.WriteString(w, `{"access_token":"connectivity-access","token_type":"Bearer","expires_in":3600}`); err != nil {
							t.Error(err)
						}
						return
					}
					reads++
					wantAuth := ""
					if mode == "client-credentials" {
						wantAuth = "Bearer connectivity-access"
					}
					if r.Method != http.MethodGet || r.URL.Path != path || r.Header.Get("Authorization") != wantAuth {
						t.Errorf("request %s %s authentication=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
					}
					if _, err := io.WriteString(w, `{"cursor":{"hasMore":false,"data":[]}}`); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				s := &connection.Settings{}
				args := []string{"--config-dir", t.TempDir(), "--auth-mode", mode, flag, server.URL + "/prefix"}
				if mode == "client-credentials" {
					args = append(args, "--client-id", "automation", "--token-url", server.URL+"/token", "--scopes", "connectivity:read")
				}
				root := commandWith(s, args...)
				root.RunE = func(cmd *cobra.Command, _ []string) error {
					client, err := s.Client(cmd.Context(), cmd, "connectivity")
					if err != nil {
						return err
					}
					_, err = client.Do(cmd.Context(), http.MethodGet, "/connectorinstances", nil, nil, nil)
					return err
				}
				if err := root.ExecuteContext(t.Context()); err != nil {
					t.Fatal(err)
				}
				wantTokens := 0
				if mode == "client-credentials" {
					wantTokens = 1
				}
				if reads != 1 || tokens != wantTokens {
					t.Fatalf("reads=%d tokens=%d", reads, tokens)
				}
			})
		}
	}
}

func TestConnectivityURLPersistenceAndOverrides(t *testing.T) {
	dir := t.TempDir()
	entry := connection.NewEntry(connection.Options{AuthMode: "none", ConnectivityURL: "http://localhost:8080/saved"})
	if err := connection.Save(dir, connection.Store{Active: "local", Connections: map[string]connection.Entry{"local": entry}}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FCTL_CONNECTIVITY_URL", "http://localhost:8080/env")
	s := &connection.Settings{}
	root := commandWith(s, "--config-dir", dir, "--connectivity-url", "http://localhost:8080/flag")
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		options, _, _, _, err := s.Resolve(cmd)
		if err != nil {
			return err
		}
		if options.ConnectivityURL != "http://localhost:8080/flag" {
			t.Fatal("flag did not override saved and env endpoints")
		}
		return connection.Validate(options)
	}
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	store, err := connection.Load(dir)
	if err != nil || store.Connections["local"].Options.ConnectivityURL != entry.Options.ConnectivityURL {
		t.Fatal("one-off flag changed saved endpoint")
	}
	//nolint:gosec // These are rejection fixtures, not credentials used by the CLI.
	for _, options := range []connection.Options{
		{AuthMode: "cloud", ConnectivityURL: "https://service.example"},
		{AuthMode: "client-credentials", ClientID: "client", TokenURL: "https://auth.example/token", ConnectivityURL: "http://service.example"},
		{AuthMode: "none", ConnectivityURL: "https://user:secret@service.example"},
	} {
		if err := connection.Validate(options); err == nil || strings.TrimSpace(err.Error()) == "" {
			t.Fatal("accepted invalid endpoint/auth combination")
		}
	}
}
