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
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/connection"
)

const (
	cliMembershipToken        = "cli-membership-secret"
	cliRefreshToken           = "cli-refresh-secret" //nolint:gosec // Synthetic OAuth2 fixture token used only to test secret redaction.
	cliDeviceCode             = "cli-device-secret"
	cliRotatedMembershipToken = "cli-rotated-membership-secret" //nolint:gosec // Synthetic Membership token for the cancellation regression.
	cliRotatedRefreshToken    = "cli-rotated-refresh-secret"    //nolint:gosec // Synthetic refresh token for the cancellation regression.
)

// Each execution creates a fresh root command; only the private config survives.
func TestCloudCLILifecycle(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t)
	dir := t.TempDir()
	f.run(t, dir, "profiles", "add", "cloud", "--auth-mode", "cloud", "--issuer", f.issuer(), "--organization", "org", "--stack", "stack")
	f.assertConnectionViews(t, dir, false)
	out, stderr := f.run(t, dir, "login")
	assertCLICloudJSON(t, out, `{"loggedIn":true}`)
	f.assertDeviceInstructions(t, out, stderr)
	f.assertRootSession(t, dir)
	f.assertCounts(t, 1, 1, 0, 0, 0)
	f.assertConnectionViews(t, dir, true)
	out, stderr = f.run(t, dir, "ledger", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	f.assertDeviceInstructions(t, out, stderr)
	f.assertTargetSaved(t, dir, "org", "stack")
	f.assertCounts(t, 2, 2, 1, 1, 0)
	out, stderr = f.run(t, dir, "auth", "clients", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	if stderr != "" {
		t.Fatal("cached target requested authorization again")
	}
	out, stderr = f.run(t, dir, "ledger", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	if stderr != "" {
		t.Fatal("cached target requested authorization again")
	}
	f.assertCounts(t, 2, 2, 1, 2, 1)
	f.assertConnectionViews(t, dir, true)
	out, _ = f.run(t, dir, "logout")
	assertCLICloudJSON(t, out, `{"loggedIn":false}`)
	store := readCLICloudStore(t, dir)
	if store.Active != "cloud" || store.Connections["cloud"].Session != nil {
		t.Fatal("logout did not clear the root session and targets")
	}
	f.assertNoSecrets(t, string(readCLICloudData(t, dir)))
	f.assertConnectionViews(t, dir, false)
	f.assertLoggedOut(t, dir)
}

func TestCloudCLILoginWithoutProfile(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t)
	dir := t.TempDir()
	out, stderr := f.run(t, dir, "login", "--issuer", f.issuer())
	assertCLICloudJSON(t, out, `{"loggedIn":true}`)
	f.assertDeviceInstructions(t, out, stderr)
	f.assertRootSession(t, dir)
	entry := readCLICloudStore(t, dir).Connections["cloud"]
	if entry.Options.AuthMode != "cloud" || entry.Options.Issuer != f.issuer() || entry.Options.Organization != "" || entry.Options.Stack != "" {
		t.Fatal("login did not create a target-free Cloud profile")
	}
	f.assertConnectionViews(t, dir, true)
	out, _ = f.run(t, dir, "ledger", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	f.assertTargetSaved(t, dir, "org", "stack")
	f.assertCounts(t, 2, 2, 1, 1, 0)
}

func TestCloudCLITargetSelectionAndCache(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t, true)
	dir := t.TempDir()
	f.run(t, dir, "login", "--issuer", f.issuer())
	for _, flags := range [][]string{nil, {"--organization", "org"}, {"--organization", "missing", "--stack", "stack"}, {"--organization", "other", "--stack", "stack"}} {
		f.assertTargetRefused(t, dir, flags)
	}
	f.assertCounts(t, 1, 1, 0, 0, 0)
	for _, target := range [][2]string{{"other", "otherstack"}, {"org", "second"}, {"other", "otherstack"}} {
		out, _ := f.run(t, dir, "--organization", target[0], "--stack", target[1], "ledger", "list")
		assertCLICloudJSON(t, out, `{"data":[]}`)
		f.assertTargetSaved(t, dir, target[0], target[1])
	}
	f.assertCounts(t, 3, 3, 2, 3, 0)
	f.assertRootSession(t, dir)
	entry := readCLICloudStore(t, dir).Connections["cloud"]
	if len(entry.Session.Targets) != 2 || entry.Options.Organization != "" || entry.Options.Stack != "" {
		t.Fatal("target override replaced defaults or lost cached targets")
	}
	f.assertTargetRefused(t, dir, nil)
	f.assertConnectionViews(t, dir, true)
}

func TestCloudCLISavedTargetDefaults(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t, true)
	dir := t.TempDir()
	f.run(t, dir, "profiles", "add", "cloud", "--auth-mode", "cloud", "--issuer", f.issuer(), "--organization", "org", "--stack", "stack")
	f.run(t, dir, "login")
	f.run(t, dir, "ledger", "list")
	f.assertTargetSaved(t, dir, "org", "stack")
	f.run(t, dir, "--organization", "other", "--stack", "otherstack", "ledger", "list")
	f.run(t, dir, "ledger", "list")
	f.assertCounts(t, 3, 3, 2, 3, 0)
	f.assertConnectionViews(t, dir, true)
	entry := readCLICloudStore(t, dir).Connections["cloud"]
	if entry.Options.Organization != "org" || entry.Options.Stack != "stack" {
		t.Fatal("override mutated saved target defaults")
	}
}

func TestCloudCLITargetRefreshPreservesOtherTargets(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t, true)
	dir := t.TempDir()
	f.run(t, dir, "login", "--issuer", f.issuer())
	f.run(t, dir, "--organization", "org", "--stack", "stack", "ledger", "list")
	f.run(t, dir, "--organization", "other", "--stack", "otherstack", "ledger", "list")
	if err := connection.Update(t.Context(), dir, func(store *connection.Store) error {
		target := findCLICloudTarget(t, store.Connections["cloud"], "org", "stack")
		target.MembershipToken.Expiry = time.Now().Add(-time.Hour)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	out, stderr := f.run(t, dir, "--organization", "org", "--stack", "stack", "ledger", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	if stderr != "" {
		t.Fatal("refresh repeated device authorization")
	}
	f.assertTargetSaved(t, dir, "org", "stack")
	f.run(t, dir, "--organization", "other", "--stack", "otherstack", "ledger", "list")
	f.assertTargetSaved(t, dir, "other", "otherstack")
	f.assertCounts(t, 3, 3, 3, 4, 0)
	if f.targets[0].refreshes.Load() != 1 || f.targets[2].refreshes.Load() != 0 {
		t.Fatal("refresh affected the wrong target")
	}
	f.assertRootSession(t, dir)
}

func TestCloudCLIRefreshSavedAfterCancellation(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t)
	dir := t.TempDir()
	f.run(t, dir, "login", "--issuer", f.issuer())
	f.run(t, dir, "ledger", "list")
	if err := connection.Update(t.Context(), dir, func(store *connection.Store) error {
		for _, target := range store.Connections["cloud"].Session.Targets {
			target.MembershipToken.Expiry = time.Now().Add(-time.Hour)
		}
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
	root.SetArgs([]string{"--no-browser", "--config-dir", dir, "--profile", "cloud", "ledger", "list"})
	err := root.ExecuteContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("command error = %v, want cancellation during Stack exchange", err)
	}
	f.assertNoSecrets(t, stdout.String()+stderr.String()+err.Error())
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Error("canceled command produced output")
	}
	f.assertCounts(t, 2, 2, 2, 1, 0)
	target := f.targets[0]
	if target.refreshes.Load() != 1 {
		t.Fatal("target Membership was not refreshed exactly once")
	}
	entry := readCLICloudStore(t, dir).Connections["cloud"]
	if entry.Revision == before {
		t.Error("rotated credentials were not saved with a new revision")
	}
	saved := findCLICloudTarget(t, entry, "org", "stack")
	token := saved.MembershipToken
	if token == nil || token.RefreshToken != cliRotatedRefreshToken || token.AccessToken != cliRotatedMembershipToken {
		t.Fatal("cancellation lost rotated target credentials on disk")
	}
	if token.TokenType != "Bearer" || !token.Expiry.After(time.Now()) || saved.StackToken != nil {
		t.Fatal("invalid target tokens after canceled exchange")
	}
	f.assertRootSession(t, dir)
}

func TestCloudCLILoginFailurePreservesConfiguration(t *testing.T) {
	t.Parallel()
	for _, profile := range []string{"fresh", "fresh explicit", "existing issuer override", "existing target change"} {
		for _, mode := range []string{"failure", "cancellation"} {
			t.Run(profile+"/"+mode, func(t *testing.T) { t.Parallel(); testCLICloudFailedLogin(t, profile, mode) })
		}
	}
}

func testCLICloudFailedLogin(t *testing.T, profile, mode string) {
	t.Helper()
	original, provider, dir, args := prepareCLICloudFailedLogin(t, profile)
	before := snapshotCLICloudConfiguration(t, dir)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if mode == "failure" {
		provider.failDevice.Store(true)
	} else {
		provider.cancelPoll.Store(&cancel)
	}
	devices, polls := provider.devices.Load(), provider.polls.Load()
	out, stderr, err := executeCLICloudContext(ctx, cmd.NewRootCommand(), append([]string{"--no-browser", "--config-dir", dir}, args...))
	if err == nil {
		t.Fatal("failed login unexpectedly succeeded")
	}
	if mode == "cancellation" && !errors.Is(err, context.Canceled) {
		t.Fatalf("login error = %v, want cancellation", err)
	}
	original.assertNoSecrets(t, out+stderr+err.Error())
	provider.assertNoSecrets(t, out+stderr+err.Error())
	if out != "" {
		t.Error("failed login printed success JSON")
	}
	expectedPolls := polls
	if mode == "cancellation" {
		expectedPolls++
	}
	if provider.devices.Load() != devices+1 || provider.polls.Load() != expectedPolls {
		t.Fatal("failure did not occur during the intended authentication phase")
	}
	assertCLICloudConfigurationUnchanged(t, dir, before)
}

func prepareCLICloudFailedLogin(t *testing.T, profile string) (*cliCloudFixture, *cliCloudFixture, string, []string) {
	t.Helper()
	f := newCLICloudFixture(t, true)
	dir := t.TempDir()
	args := []string{"login", "--issuer", f.issuer()}
	if profile == "fresh" {
		return f, f, dir, args
	}
	if profile == "fresh explicit" {
		return f, f, dir, append(args, "--profile", "new-cloud")
	}
	f.run(t, dir, "login", "--issuer", f.issuer(), "--organization", "org", "--stack", "stack")
	f.run(t, dir, "ledger", "list")
	f.assertTargetSaved(t, dir, "org", "stack")
	f.run(t, dir, "profiles", "add", "local", "--auth-mode", "none", "--ledger-url", f.server.URL+"/unused")
	f.run(t, dir, "profiles", "use", "local")
	args = []string{"login", "--profile", "cloud"}
	if profile == "existing issuer override" {
		other := newCLICloudFixture(t, true)
		return f, other, dir, append(args, "--issuer", other.issuer())
	}
	return f, f, dir, append(args, "--organization", "other", "--stack", "otherstack")
}

func executeCLICloudContext(ctx context.Context, root *cobra.Command, args []string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	return stdout.String(), stderr.String(), err
}

// nil records absence; a failed first login must not create connections.json.
func snapshotCLICloudConfiguration(t *testing.T, dir string) []byte {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, "connections.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return readCLICloudData(t, dir)
}

func assertCLICloudConfigurationUnchanged(t *testing.T, dir string, before []byte) {
	t.Helper()
	after := snapshotCLICloudConfiguration(t, dir)
	if (before == nil) != (after == nil) || !bytes.Equal(before, after) {
		t.Fatal("failed login changed connections.json, active selection or saved credentials")
	}
}

func TestCloudCLIInvalidLoginOptionsDoNotCreateProfile(t *testing.T) {
	t.Parallel()
	for _, flags := range [][]string{{"--output", "yaml"}, {"--timeout", "0s"}, {"--timeout", "-1s"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			t.Parallel()
			f := newCLICloudFixture(t)
			dir := t.TempDir()
			args := append([]string{"--no-browser", "--config-dir", dir, "login", "--issuer", f.issuer()}, flags...)
			out, stderr, err := executeRoot(t, args...)
			if err == nil {
				t.Fatal("invalid login options accepted")
			}
			if out != "" || stderr != "" || f.requests.Load() != 0 {
				t.Fatal("invalid options started authentication")
			}
			assertCLICloudConfigurationUnchanged(t, dir, nil)
		})
	}
}

type cliCloudTarget struct {
	organization, stack, idToken                     string
	membership, refreshToken, stackToken, deviceCode string
	refreshes                                        atomic.Int32
}

type cliCloudFixture struct {
	server                                                *httptest.Server
	idToken                                               string
	targets                                               []*cliCloudTarget
	devices, polls, exchanges, ledgers, clients, requests atomic.Int32
	cancelExchange                                        atomic.Pointer[context.CancelFunc]
	failDevice                                            atomic.Bool
	cancelPoll                                            atomic.Pointer[context.CancelFunc]
}

func newCLICloudFixture(t *testing.T, multiple ...bool) *cliCloudFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "cli-key"))
	if err != nil {
		t.Fatal(err)
	}
	f := &cliCloudFixture{targets: []*cliCloudTarget{newCLICloudTarget("org", "stack")}}
	if len(multiple) != 0 && multiple[0] {
		f.targets = append(f.targets, newCLICloudTarget("org", "second"), newCLICloudTarget("other", "otherstack"))
	}
	mux := http.NewServeMux()
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.requests.Add(1); mux.ServeHTTP(w, r) }))
	t.Cleanup(f.server.Close)
	f.idToken = f.signIdentity(t, signer, f.targets, "root")
	for _, target := range f.targets {
		target.idToken = f.signIdentity(t, signer, []*cliCloudTarget{target}, target.organization+"/"+target.stack)
	}
	f.registerDiscovery(t, mux, &key.PublicKey)
	mux.HandleFunc("POST /membership/device", func(w http.ResponseWriter, r *http.Request) { f.device(t, w, r) })
	mux.HandleFunc("POST /membership/token", func(w http.ResponseWriter, r *http.Request) { f.membershipToken(t, w, r) })
	for _, target := range f.targets {
		f.registerTarget(t, mux, target)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Cloud request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	return f
}

func newCLICloudTarget(org, stack string) *cliCloudTarget {
	return &cliCloudTarget{organization: org, stack: stack, membership: "scoped-" + stack + "-secret", refreshToken: "refresh-" + stack + "-secret", stackToken: "api-" + stack + "-secret", deviceCode: "device-" + stack + "-secret"}
}

func (f *cliCloudFixture) issuer() string { return f.server.URL + "/membership" }

func (f *cliCloudFixture) signIdentity(t *testing.T, signer jose.Signer, targets []*cliCloudTarget, identity string) string {
	t.Helper()
	organizations := map[string][]any{}
	for _, target := range targets {
		organizations[target.organization] = append(organizations[target.organization], map[string]any{"id": target.stack, "uri": f.server.URL + "/" + target.stack, "scopes": []string{"stack:Read", "stack:Write"}})
	}
	var orgs []any
	for org, stacks := range organizations {
		orgs = append(orgs, map[string]any{"id": org, "stacks": stacks})
	}
	raw, err := jwt.Signed(signer).Claims(map[string]any{"iss": f.issuer(), "aud": "fctl", "sub": "cli-user", "jti": identity, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "org": orgs}).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (f *cliCloudFixture) registerDiscovery(t *testing.T, mux *http.ServeMux, key *rsa.PublicKey) {
	mux.HandleFunc("GET /membership/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeCLICloudJSON(t, w, map[string]any{"issuer": f.issuer(), "token_endpoint": f.issuer() + "/token", "device_authorization_endpoint": f.issuer() + "/device", "jwks_uri": f.issuer() + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	})
	mux.HandleFunc("GET /membership/keys", func(w http.ResponseWriter, _ *http.Request) {
		writeCLICloudJSON(t, w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: key, KeyID: "cli-key", Algorithm: "RS256", Use: "sig"}}})
	})
}

func (f *cliCloudFixture) registerTarget(t *testing.T, mux *http.ServeMux, target *cliCloudTarget) {
	issuer := f.server.URL + "/" + target.stack + "/api/auth"
	mux.HandleFunc("GET /"+target.stack+"/api/auth/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeCLICloudJSON(t, w, map[string]any{"issuer": issuer, "token_endpoint": issuer + "/token"})
	})
	mux.HandleFunc("POST /"+target.stack+"/api/auth/token", func(w http.ResponseWriter, r *http.Request) { f.exchange(t, w, r, target) })
	mux.HandleFunc("GET /"+target.stack+"/api/ledger/v3/{$}", func(w http.ResponseWriter, r *http.Request) { f.service(t, w, r, target, &f.ledgers) })
	mux.HandleFunc("GET /"+target.stack+"/api/auth/clients", func(w http.ResponseWriter, r *http.Request) { f.service(t, w, r, target, &f.clients) })
}

func (f *cliCloudFixture) device(t *testing.T, w http.ResponseWriter, r *http.Request) {
	f.devices.Add(1)
	if !parseCLICloudForm(t, w, r) {
		return
	}
	assertCLICloudBasic(t, r)
	assertCLICloudForm(t, r, map[string]string{"client_id": "fctl"})
	if f.failDevice.Load() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		writeCLICloudJSON(t, w, map[string]string{"error": "access_denied", "error_description": cliMembershipToken})
		return
	}
	code := cliDeviceCode
	if resource := r.PostForm.Get("resource"); resource != "" {
		target := f.targetForResource(t, resource)
		if target == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assertCLICloudForm(t, r, map[string]string{"organization_id": target.organization, "id_token_hint": f.idToken, "scope": "openid offline_access accesses"})
		code = target.deviceCode
	} else {
		assertCLICloudForm(t, r, map[string]string{"scope": "openid offline_access accesses on_behalf", "prompt": "no-org"})
		for _, name := range []string{"organization_id", "resource", "id_token_hint"} {
			if _, exists := r.PostForm[name]; exists {
				t.Errorf("root identity device request included %s", name)
			}
		}
	}
	writeCLICloudJSON(t, w, map[string]any{"device_code": code, "user_code": "ABCD-EFGH", "verification_uri": f.server.URL + "/verify", "expires_in": 600, "interval": 1})
}

func (f *cliCloudFixture) targetForResource(t *testing.T, resource string) *cliCloudTarget {
	t.Helper()
	for _, target := range f.targets {
		if resource == "stack://"+target.organization+"/"+target.stack+"|stack:Read stack:Write" {
			return target
		}
	}
	t.Error("unknown scoped resource")
	return nil
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
	assertCLICloudForm(t, r, map[string]string{"grant_type": "urn:ietf:params:oauth:grant-type:device_code"})
	if rejectCLICloudPollScope(t, w, r) {
		return
	}
	if cancel := f.cancelPoll.Load(); cancel != nil {
		(*cancel)()
		<-r.Context().Done()
		return
	}
	membership, refresh, id := cliMembershipToken, cliRefreshToken, f.idToken
	if r.PostForm.Get("device_code") != cliDeviceCode {
		target := f.targetForResource(t, r.PostForm.Get("resource"))
		if target == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assertCLICloudForm(t, r, map[string]string{"device_code": target.deviceCode})
		membership, refresh, id = target.membership, target.refreshToken, target.idToken
	} else if _, exists := r.PostForm["resource"]; exists {
		t.Error("root poll included resource")
	}
	writeCLICloudJSON(t, w, map[string]any{"access_token": membership, "refresh_token": refresh, "id_token": id, "token_type": "Bearer", "expires_in": 3600})
}

func rejectCLICloudPollScope(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	if len(r.PostForm["scope"]) == 0 {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	writeCLICloudJSON(t, w, map[string]string{
		"error":             "server_error",
		"error_description": "requested scope " + r.PostForm.Get("scope") + " not granted",
	})
	return true
}

func (f *cliCloudFixture) refresh(t *testing.T, w http.ResponseWriter, r *http.Request) {
	for _, target := range f.targets {
		if r.PostForm.Get("refresh_token") == target.refreshToken {
			target.refreshes.Add(1)
			writeCLICloudJSON(t, w, map[string]any{"access_token": cliRotatedMembershipToken, "refresh_token": cliRotatedRefreshToken, "token_type": "Bearer", "expires_in": 3600})
			return
		}
	}
	t.Error("refresh used root or unknown target token")
	w.WriteHeader(http.StatusUnauthorized)
}

func (f *cliCloudFixture) exchange(t *testing.T, w http.ResponseWriter, r *http.Request, target *cliCloudTarget) {
	f.exchanges.Add(1)
	if !parseCLICloudForm(t, w, r) {
		return
	}
	assertion := target.membership
	if target.refreshes.Load() != 0 {
		assertion = cliRotatedMembershipToken
	}
	assertCLICloudForm(t, r, map[string]string{"grant_type": "urn:ietf:params:oauth:grant-type:jwt-bearer", "assertion": assertion, "scope": "openid email"})
	if cancel := f.cancelExchange.Load(); cancel != nil {
		(*cancel)()
		<-r.Context().Done()
		return
	}
	writeCLICloudJSON(t, w, map[string]any{"access_token": target.stackToken, "token_type": "Bearer", "expires_in": 3600})
}

func (f *cliCloudFixture) service(t *testing.T, w http.ResponseWriter, r *http.Request, target *cliCloudTarget, counter *atomic.Int32) {
	counter.Add(1)
	if r.Header.Get("Authorization") != "Bearer "+target.stackToken {
		t.Error("service request used another target or root token")
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
	out, stderr, err := executeRoot(t, append([]string{"--no-browser", "--config-dir", dir}, args...)...)
	f.assertNoSecrets(t, out+stderr)
	if err != nil {
		f.assertNoSecrets(t, err.Error())
		t.Fatalf("%v failed: %v", args, err)
	}

	if !json.Valid([]byte(out)) {
		t.Fatalf("%v did not produce JSON", args)
	}
	return out, stderr
}

func (f *cliCloudFixture) assertNoSecrets(t *testing.T, output string) {
	t.Helper()
	for _, target := range f.targets {
		for _, secret := range []string{target.membership, target.refreshToken, target.stackToken, target.deviceCode, target.idToken} {
			if strings.Contains(output, secret) {
				t.Error("CLI output exposed target credentials")
			}
		}
	}
	for _, secret := range []string{cliMembershipToken, cliRefreshToken, cliDeviceCode, cliRotatedMembershipToken, cliRotatedRefreshToken, f.idToken} {
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

func (f *cliCloudFixture) assertRootSession(t *testing.T, dir string) {
	t.Helper()
	store := readCLICloudStore(t, dir)
	s := store.Connections["cloud"].Session
	if s == nil || s.MembershipToken == nil {
		t.Fatal("login did not persist root Membership session")
	}
	if store.Active != "cloud" || s.IDToken != f.idToken || s.StackToken != nil || s.StackURL != "" {
		t.Fatal("saved root identity contains scoped credentials")
	}
	if s.Options.Issuer != f.issuer() || s.Options.ClientID != "fctl" || s.Options.Organization != "" || s.Options.Stack != "" {
		t.Fatal("root session lost target-free identity options")
	}
	if s.MembershipToken.AccessToken != cliMembershipToken || s.MembershipToken.RefreshToken != cliRefreshToken || !s.MembershipToken.Expiry.After(time.Now()) {
		t.Fatal("root credentials were lost or replaced by target credentials")
	}
}

func findCLICloudTarget(t *testing.T, entry connection.Entry, organization, stack string) *cloud.Session {
	t.Helper()
	if entry.Session != nil {
		for _, target := range entry.Session.Targets {
			if target != nil && target.Options.Organization == organization && target.Options.Stack == stack {
				return target
			}
		}
	}
	t.Fatal("requested target was not persisted")
	return nil
}

func (f *cliCloudFixture) assertTargetSaved(t *testing.T, dir, org, stack string) {
	t.Helper()
	saved := findCLICloudTarget(t, readCLICloudStore(t, dir).Connections["cloud"], org, stack)
	target := f.targetForResource(t, "stack://"+org+"/"+stack+"|stack:Read stack:Write")
	expectedMembership, expectedRefresh := target.membership, target.refreshToken
	if target.refreshes.Load() != 0 {
		expectedMembership, expectedRefresh = cliRotatedMembershipToken, cliRotatedRefreshToken
	}
	if saved.MembershipToken == nil || saved.StackToken == nil {
		t.Fatal("target credentials not persisted")
	}
	if saved.IDToken != target.idToken || saved.StackURL != f.server.URL+"/"+stack || saved.MembershipToken.AccessToken != expectedMembership || saved.MembershipToken.RefreshToken != expectedRefresh {
		t.Fatal("saved target identity or Membership credentials differ")
	}
	if saved.StackToken.AccessToken != target.stackToken || !saved.StackToken.Expiry.After(time.Now()) {
		t.Fatal("exchanged target token was not persisted")
	}
}

func (f *cliCloudFixture) assertDeviceInstructions(t *testing.T, out, stderr string) {
	t.Helper()
	if !strings.Contains(stderr, f.server.URL+"/verify") || !strings.Contains(stderr, "ABCD-EFGH") {
		t.Fatal("device URL and user code missing from stderr")
	}
	if strings.Contains(out, "/verify") || strings.Contains(out, "ABCD-EFGH") {
		t.Fatal("device instructions leaked into JSON stdout")
	}
}

func (f *cliCloudFixture) assertTargetRefused(t *testing.T, dir string, flags []string) {
	t.Helper()
	before := [3]int32{f.devices.Load(), f.exchanges.Load(), f.ledgers.Load()}
	args := append([]string{"--no-browser", "--config-dir", dir}, flags...)
	out, stderr, err := executeRoot(t, append(args, "ledger", "list")...)
	if err == nil {
		t.Fatal("ambiguous or invalid target accepted")
	}
	f.assertNoSecrets(t, out+stderr+err.Error())
	after := [3]int32{f.devices.Load(), f.exchanges.Load(), f.ledgers.Load()}
	if before != after || out != "" || stderr != "" {
		t.Fatal("invalid target authorized a device or contacted services")
	}
}

func (f *cliCloudFixture) assertConnectionViews(t *testing.T, dir string, loggedIn bool) {
	t.Helper()
	out, _ := f.run(t, dir, "profiles", "show")
	var options connection.Options
	if err := json.Unmarshal([]byte(out), &options); err != nil {
		t.Fatal(err)
	}
	saved := readCLICloudStore(t, dir).Connections["cloud"].Options
	if options.AuthMode != "cloud" || options.Issuer != f.server.URL+"/membership" || options.Organization != saved.Organization || options.Stack != saved.Stack {
		t.Fatal("connection show lost saved Cloud settings")
	}
	out, _ = f.run(t, dir, "profiles", "list")
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
		out, stderr, err := executeRoot(t, append([]string{"--no-browser", "--config-dir", dir}, args...)...)
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
