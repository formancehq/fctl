package cmd_test

import (
	"bytes"
	"context"
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

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/connection"
)

const (
	cliMembershipToken        = "cli-membership-secret"
	cliRefreshToken           = "cli-refresh-secret" //nolint:gosec // Synthetic OAuth2 fixture token used only to test secret redaction.
	cliStackToken             = "cli-stack-secret"
	cliDeviceCode             = "cli-device-secret"
	cliResource               = "stack://org/stack|stack:Read stack:Write"
	cliRotatedMembershipToken = "cli-rotated-membership-secret" //nolint:gosec // Synthetic Membership token for the cancellation regression.
	cliRotatedRefreshToken    = "cli-rotated-refresh-secret"    //nolint:gosec // Synthetic refresh token for the cancellation regression.
)

// Each execution creates a fresh root command; only the private config survives.
func TestCloudCLILifecycle(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t)
	dir := t.TempDir()
	f.run(t, dir, "connections", "add", "cloud", "--auth-mode", "cloud", "--issuer", f.server.URL+"/membership", "--organization", "org", "--stack", "stack")
	readCLICloudStore(t, dir)
	f.assertConnectionViews(t, dir, false)

	out, stderr := f.run(t, dir, "login")
	assertCLICloudJSON(t, out, `{"loggedIn":true}`)
	if !strings.Contains(stderr, f.server.URL+"/verify") || !strings.Contains(stderr, "ABCD-EFGH") {
		t.Fatal("login did not display the device URL and user code on stderr")
	}
	if strings.Contains(out, "/verify") || strings.Contains(out, "ABCD-EFGH") {
		t.Fatal("device instructions leaked into JSON stdout")
	}
	f.assertSavedSession(t, dir, false)
	f.assertCounts(t, 1, 1, 0, 0, 0)
	f.assertConnectionViews(t, dir, true)

	out, _ = f.run(t, dir, "ledger", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	f.assertSavedSession(t, dir, true)
	f.assertCounts(t, 1, 1, 1, 1, 0)
	out, _ = f.run(t, dir, "auth", "clients", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	out, _ = f.run(t, dir, "ledger", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	f.assertCounts(t, 1, 1, 1, 2, 1)
	f.assertConnectionViews(t, dir, true)

	out, _ = f.run(t, dir, "logout")
	assertCLICloudJSON(t, out, `{"loggedIn":false}`)
	store := readCLICloudStore(t, dir)
	if store.Active != "cloud" || store.Connections["cloud"].Session != nil {
		t.Fatal("logout did not clear the session while preserving the selected connection")
	}
	data := readCLICloudData(t, dir)
	f.assertNoSecrets(t, string(data))
	f.assertConnectionViews(t, dir, false)
	f.assertLoggedOut(t, dir)
}

func TestCloudCLIRefreshSavedAfterCancellation(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t)
	dir := t.TempDir()
	f.run(t, dir, "connections", "add", "cloud", "--auth-mode", "cloud", "--issuer", f.server.URL+"/membership", "--organization", "org", "--stack", "stack")
	f.run(t, dir, "login")
	f.assertSavedSession(t, dir, false)
	if err := connection.Update(t.Context(), dir, func(store *connection.Store) error {
		store.Connections["cloud"].Session.MembershipToken.Expiry = time.Now().Add(-time.Hour)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before := readCLICloudStore(t, dir).Connections["cloud"].Revision
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	f.cancelExchange.Store(&cancel)
	root := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--config-dir", dir, "--connection", "cloud", "ledger", "list"})
	err := root.ExecuteContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("command error = %v, want cancellation during Stack exchange", err)
	}
	f.assertNoSecrets(t, stdout.String()+stderr.String()+err.Error())
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Error("canceled command produced output")
	}
	f.assertCounts(t, 1, 1, 1, 0, 0)
	if f.refreshes.Load() != 1 {
		t.Fatal("command did not refresh Membership exactly once")
	}
	entry := readCLICloudStore(t, dir).Connections["cloud"]
	if entry.Revision == before {
		t.Error("rotated credentials were not saved with a new revision")
	}
	assertCLICloudRotatedSession(t, entry)
}

func assertCLICloudRotatedSession(t *testing.T, entry connection.Entry) {
	t.Helper()
	if entry.Session == nil || entry.Session.MembershipToken == nil {
		t.Fatal("cancellation lost the Membership session")
	}
	token := entry.Session.MembershipToken
	if token.RefreshToken != cliRotatedRefreshToken || token.AccessToken != cliRotatedMembershipToken {
		t.Error("cancellation lost the rotated Membership credentials on disk")
	}
	if token.TokenType != "Bearer" || !token.Expiry.After(time.Now()) {
		t.Error("saved rotated token has invalid type or expiry")
	}
	if entry.Session.StackToken != nil {
		t.Error("canceled exchange persisted a Stack token")
	}
}

type cliCloudFixture struct {
	server                                                *httptest.Server
	idToken                                               string
	devices, polls, exchanges, ledgers, clients, requests atomic.Int32
	refreshes                                             atomic.Int32
	cancelExchange                                        atomic.Pointer[context.CancelFunc]
}

func newCLICloudFixture(t *testing.T) *cliCloudFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "cli-key"))
	if err != nil {
		t.Fatal(err)
	}
	f := &cliCloudFixture{}
	mux := http.NewServeMux()
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(f.server.Close)
	f.idToken, err = jwt.Signed(signer).Claims(map[string]any{
		"iss": f.server.URL + "/membership", "aud": "fctl", "sub": "cli-user",
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		"org": []any{map[string]any{"id": "org", "stacks": []any{map[string]any{
			"id": "stack", "uri": f.server.URL + "/stack", "scopes": []string{"stack:Read", "stack:Write"},
		}}}},
	}).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	f.registerDiscovery(t, mux, &key.PublicKey)
	mux.HandleFunc("POST /membership/device", func(w http.ResponseWriter, r *http.Request) { f.device(t, w, r) })
	mux.HandleFunc("POST /membership/token", func(w http.ResponseWriter, r *http.Request) { f.membershipToken(t, w, r) })
	mux.HandleFunc("POST /stack/api/auth/token", func(w http.ResponseWriter, r *http.Request) { f.exchange(t, w, r) })
	mux.HandleFunc("GET /stack/api/ledger/v3/{$}", func(w http.ResponseWriter, r *http.Request) { f.service(t, w, r, &f.ledgers) })
	mux.HandleFunc("GET /stack/api/auth/clients", func(w http.ResponseWriter, r *http.Request) { f.service(t, w, r, &f.clients) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Cloud request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	return f
}

func (f *cliCloudFixture) registerDiscovery(t *testing.T, mux *http.ServeMux, key *rsa.PublicKey) {
	mux.HandleFunc("GET /membership/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeCLICloudJSON(t, w, map[string]any{
			"issuer": f.server.URL + "/membership", "token_endpoint": f.server.URL + "/membership/token",
			"device_authorization_endpoint": f.server.URL + "/membership/device", "jwks_uri": f.server.URL + "/membership/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("GET /membership/keys", func(w http.ResponseWriter, _ *http.Request) {
		writeCLICloudJSON(t, w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: key, KeyID: "cli-key", Algorithm: "RS256", Use: "sig"}}})
	})
	mux.HandleFunc("GET /stack/api/auth/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeCLICloudJSON(t, w, map[string]any{"issuer": f.server.URL + "/stack/api/auth", "token_endpoint": f.server.URL + "/stack/api/auth/token"})
	})
}

func (f *cliCloudFixture) device(t *testing.T, w http.ResponseWriter, r *http.Request) {
	f.devices.Add(1)
	if !parseCLICloudForm(t, w, r) {
		return
	}
	assertCLICloudBasic(t, r)
	assertCLICloudForm(t, r, map[string]string{"client_id": "fctl", "organization_id": "org", "resource": cliResource, "scope": "openid offline_access"})
	writeCLICloudJSON(t, w, map[string]any{"device_code": cliDeviceCode, "user_code": "ABCD-EFGH", "verification_uri": f.server.URL + "/verify", "expires_in": 600, "interval": 1})
}

func (f *cliCloudFixture) membershipToken(t *testing.T, w http.ResponseWriter, r *http.Request) {
	if !parseCLICloudForm(t, w, r) {
		return
	}
	assertCLICloudBasic(t, r)
	if r.PostForm.Get("grant_type") == "refresh_token" {
		f.refresh(t, w, r)
		return
	}
	f.polls.Add(1)
	assertCLICloudForm(t, r, map[string]string{"grant_type": "urn:ietf:params:oauth:grant-type:device_code", "device_code": cliDeviceCode, "resource": cliResource})
	writeCLICloudJSON(t, w, map[string]any{"access_token": cliMembershipToken, "refresh_token": cliRefreshToken, "id_token": f.idToken, "token_type": "Bearer", "expires_in": 3600})
}

func (f *cliCloudFixture) refresh(t *testing.T, w http.ResponseWriter, r *http.Request) {
	f.refreshes.Add(1)
	assertCLICloudForm(t, r, map[string]string{"grant_type": "refresh_token", "refresh_token": cliRefreshToken})
	writeCLICloudJSON(t, w, map[string]any{"access_token": cliRotatedMembershipToken, "refresh_token": cliRotatedRefreshToken, "token_type": "Bearer", "expires_in": 3600})
}

func (f *cliCloudFixture) exchange(t *testing.T, w http.ResponseWriter, r *http.Request) {
	f.exchanges.Add(1)
	if !parseCLICloudForm(t, w, r) {
		return
	}
	assertion := cliMembershipToken
	if f.refreshes.Load() != 0 {
		assertion = cliRotatedMembershipToken
	}
	assertCLICloudForm(t, r, map[string]string{"grant_type": "urn:ietf:params:oauth:grant-type:jwt-bearer", "assertion": assertion, "scope": "openid email"})
	if cancel := f.cancelExchange.Load(); cancel != nil {
		(*cancel)()
		<-r.Context().Done()
		return
	}
	writeCLICloudJSON(t, w, map[string]any{"access_token": cliStackToken, "token_type": "Bearer", "expires_in": 3600})
}

func (f *cliCloudFixture) service(t *testing.T, w http.ResponseWriter, r *http.Request, counter *atomic.Int32) {
	counter.Add(1)
	if r.Header.Get("Authorization") != "Bearer "+cliStackToken {
		t.Error("service request did not use the exchanged Stack bearer")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	writeCLICloudJSON(t, w, map[string]any{"data": []any{}})
}

func parseCLICloudForm(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	return true
}

func assertCLICloudBasic(t *testing.T, r *http.Request) {
	t.Helper()
	id, secret, ok := r.BasicAuth()
	if !ok || id != "fctl" || secret != "" {
		t.Error("expected public client Basic authentication for fctl")
	}
}

func assertCLICloudForm(t *testing.T, r *http.Request, want map[string]string) {
	t.Helper()
	for name, value := range want {
		if r.PostForm.Get(name) != value {
			t.Errorf("incorrect %s form field at %s", name, r.URL.Path)
		}
	}
}

func writeCLICloudJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}

func (f *cliCloudFixture) run(t *testing.T, dir string, args ...string) (string, string) {
	t.Helper()
	out, stderr, err := executeRoot(t, append([]string{"--config-dir", dir, "--connection", "cloud"}, args...)...)
	f.assertNoSecrets(t, out+stderr)
	if err != nil {
		t.Fatalf("%v failed: %v", args, err)
	}
	if args[0] != "login" && stderr != "" {
		t.Fatalf("%v produced unexpected stderr", args)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("%v did not produce JSON", args)
	}
	return out, stderr
}

func (f *cliCloudFixture) assertNoSecrets(t *testing.T, output string) {
	t.Helper()
	for _, secret := range []string{cliMembershipToken, cliRefreshToken, cliStackToken, cliDeviceCode, cliRotatedMembershipToken, cliRotatedRefreshToken, f.idToken} {
		if strings.Contains(output, secret) {
			t.Error("CLI output exposed a token secret")
		}
	}
}

func assertCLICloudJSON(t *testing.T, out, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal([]byte(out), &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	gotJSON, err := json.Marshal(gotValue)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(wantValue)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("JSON result = %s, want %s", gotJSON, wantJSON)
	}
}

func readCLICloudStore(t *testing.T, dir string) connection.Store {
	t.Helper()
	path := filepath.Join(dir, "connections.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions = %v, want 0600", info.Mode().Perm())
	}
	data := readCLICloudData(t, dir)
	var store connection.Store
	if err := json.Unmarshal(data, &store); err != nil {
		t.Fatal(err)
	}
	return store
}

func (f *cliCloudFixture) assertSavedSession(t *testing.T, dir string, exchanged bool) {
	t.Helper()
	store := readCLICloudStore(t, dir)
	s := store.Connections["cloud"].Session
	if s == nil || s.MembershipToken == nil {
		t.Fatal("login did not persist Membership session")
	}
	if store.Active != "cloud" || s.StackURL != f.server.URL+"/stack" || s.IDToken != f.idToken {
		t.Fatal("saved identity or stack differs from signed claims")
	}
	if s.Options.Issuer != f.server.URL+"/membership" || s.Options.ClientID != "fctl" || s.Options.Organization != "org" || s.Options.Stack != "stack" {
		t.Fatal("saved session lost Cloud options")
	}
	if s.MembershipToken.AccessToken != cliMembershipToken || s.MembershipToken.RefreshToken != cliRefreshToken || s.MembershipToken.TokenType != "Bearer" || !s.MembershipToken.Expiry.After(time.Now()) {
		t.Fatal("Membership credentials were not saved correctly")
	}
	if !exchanged {
		if s.StackToken != nil {
			t.Fatal("login unexpectedly exchanged a Stack token")
		}
		return
	}
	if s.StackToken == nil || s.StackToken.AccessToken != cliStackToken || s.StackToken.TokenType != "Bearer" || !s.StackToken.Expiry.After(time.Now()) {
		t.Fatal("exchanged Stack token was not persisted")
	}
}

func (f *cliCloudFixture) assertConnectionViews(t *testing.T, dir string, loggedIn bool) {
	t.Helper()
	out, _ := f.run(t, dir, "connections", "show")
	var options connection.Options
	if err := json.Unmarshal([]byte(out), &options); err != nil {
		t.Fatal(err)
	}
	if options.AuthMode != "cloud" || options.Issuer != f.server.URL+"/membership" || options.Organization != "org" || options.Stack != "stack" {
		t.Fatal("connection show lost saved Cloud settings")
	}
	out, _ = f.run(t, dir, "connections", "list")
	var rows []struct {
		Name     string
		Active   bool
		LoggedIn bool
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Name != "cloud" || !rows[0].Active || rows[0].LoggedIn != loggedIn {
		t.Fatal("connection list reported incorrect login state")
	}
}

func (f *cliCloudFixture) assertCounts(t *testing.T, devices, polls, exchanges, ledgers, clients int32) {
	t.Helper()
	got := [5]int32{f.devices.Load(), f.polls.Load(), f.exchanges.Load(), f.ledgers.Load(), f.clients.Load()}
	want := [5]int32{devices, polls, exchanges, ledgers, clients}
	if got != want {
		t.Fatalf("device/poll/exchange/ledger/auth requests = %v, want %v", got, want)
	}
}

func (f *cliCloudFixture) assertLoggedOut(t *testing.T, dir string) {
	t.Helper()
	before := f.requests.Load()
	for _, args := range [][]string{{"ledger", "list"}, {"auth", "clients", "list"}} {
		out, stderr, err := executeRoot(t, append([]string{"--config-dir", dir, "--connection", "cloud"}, args...)...)
		if err == nil || !strings.Contains(err.Error(), "not logged in") {
			t.Fatalf("%v did not refuse the logged-out session: %v", args, err)
		}
		f.assertNoSecrets(t, out+stderr+err.Error())
		if out != "" || stderr != "" {
			t.Error("logged-out service command produced output")
		}
	}
	if f.requests.Load() != before {
		t.Fatal("logged-out commands contacted the provider or services")
	}
}

func readCLICloudData(t *testing.T, dir string) []byte {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	data, err := root.ReadFile("connections.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
