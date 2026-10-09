package cmd_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func TestDebugLedgerKeepsJSONOnStdout(t *testing.T) {
	t.Parallel()
	const payload = `{"data":[{"name":"books","amount":123456789012345678901234567890,"metadata":{"password":"debug-metadata-private"}}],"hasMore":false}`
	server := httptest.NewServer(debugLedgerHandler(t, payload))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	cacheDistributionLedger(t, dir, pluginmanager.Target{Endpoint: server.URL + "/prefix"})
	args := []string{"--config-dir", dir, "--auth-mode", "none", "--ledger-url", server.URL + "/prefix", "ledger", "list"}
	baseline, stderr, err := executeRoot(t, args...)
	if err != nil || stderr != "" {
		t.Fatalf("without debug: stderr=%q err=%v", stderr, err)
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, []byte(payload), "", "  "); err != nil {
		t.Fatal(err)
	}
	if baseline != formatted.String()+"\n" {
		t.Fatalf("baseline changed JSON or integer precision: %s", baseline)
	}
	for _, flag := range []string{"-d", "--debug"} {
		t.Run(flag, func(t *testing.T) {
			out, trace, err := executeRoot(t, append([]string{flag}, args...)...)
			if err != nil || out != baseline {
				t.Fatalf("debug changed result: stdout=%q err=%v", out, err)
			}
			assertDebugTraceContains(t, trace, `request "GET"`, "/prefix/v3/", "response 200", "response body:", "123456789012345678901234567890", "[REDACTED]")
			assertDebugTraceHides(t, trace, "debug-metadata-private", "debug-cookie-private")
		})
	}
}

func debugLedgerHandler(t *testing.T, payload string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/prefix/_info" {
			writeCLICloudJSON(t, w, map[string]string{"version": "3.0.0"})
			return
		}
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/prefix/v3/" || r.URL.RawQuery != "pageSize=100" {
			t.Errorf("unexpected ledger request: %s %s", r.Method, r.URL.RequestURI())
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("anonymous debug request sent credentials")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "session=debug-cookie-private")
		if _, err := io.WriteString(w, payload); err != nil {
			t.Error(err)
		}
	})
}

func TestDebugClientCredentialsRedactsAuthentication(t *testing.T) {
	const clientID = "debug-client"
	const clientSecret = "debug-client-private" //nolint:gosec // Synthetic fixture credential used only to verify trace redaction.
	const accessToken = "debug-access-private"  //nolint:gosec // Synthetic fixture token used only to verify trace redaction.
	t.Setenv("FCTL_CLIENT_SECRET", clientSecret)
	var tokenCalls, ledgerCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/_info":
			writeCLICloudJSON(t, w, map[string]string{"version": "3.0.0"})
		case "/token":
			tokenCalls.Add(1)
			assertDebugClientCredentials(t, r, clientID, clientSecret)
			writeCLICloudJSON(t, w, map[string]any{"access_token": accessToken, "token_type": "Bearer", "expires_in": 3600, "refresh_token": "debug-refresh-private", "id_token": "debug-identity-private"}) //nolint:gosec // Synthetic OAuth response verifies that traces hide every token.
		case "/v3/":
			ledgerCalls.Add(1)
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer "+accessToken {
				t.Error("Ledger request lost its bearer credentials")
			}
			writeCLICloudJSON(t, w, map[string]any{"data": []any{}})
		default:
			t.Errorf("unexpected authentication route: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	cacheDistributionLedger(t, dir, pluginmanager.Target{Endpoint: server.URL})
	out, trace, err := executeRoot(t, "-d", "--config-dir", dir, "--auth-mode", "client-credentials", "--client-id", clientID, "--token-url", server.URL+"/token", "--ledger-url", server.URL, "ledger", "list")
	if err != nil {
		t.Fatal(err)
	}
	assertCLICloudJSON(t, out, `{"data":[]}`)
	assertDebugTraceContains(t, trace, `request "POST"`, "/token", "client_credentials", `request "GET"`, "/v3/", "response 200", "access_token", "[REDACTED]")
	basic := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	assertDebugTraceHides(t, trace, clientSecret, accessToken, "debug-refresh-private", "debug-identity-private", basic)
	if tokenCalls.Load() != 1 || ledgerCalls.Load() != 1 {
		t.Fatalf("debug repeated authentication or service call: token=%d ledger=%d", tokenCalls.Load(), ledgerCalls.Load())
	}
}

func assertDebugClientCredentials(t *testing.T, r *http.Request, clientID, clientSecret string) {
	t.Helper()
	id, secret, ok := r.BasicAuth()
	if r.Method != http.MethodPost || !ok || id != clientID || secret != clientSecret {
		t.Error("debug tracing changed client credentials")
	}
	if err := r.ParseForm(); err != nil {
		t.Error(err)
		return
	}
	if r.PostForm.Get("grant_type") != "client_credentials" {
		t.Error("debug tracing consumed the OAuth form")
	}
}

func TestDebugCloudAuthenticationAndServices(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t)
	dir := t.TempDir()
	cacheDistributionAuth(t, dir, pluginmanager.Target{Profile: "cloud", Organization: "org", Stack: "stack", Endpoint: f.issuer()})
	out, trace := f.run(t, dir, "--debug", "login", "--issuer", f.issuer())
	assertCLICloudJSON(t, out, `{"loggedIn":true}`)
	assertDebugTraceContains(t, trace, "/membership/.well-known/openid-configuration", "/membership/device", "/membership/token", "response 200", "device_code", "access_token", "[REDACTED]")
	f.assertNoSecrets(t, trace)
	out, trace = f.run(t, dir, "-d", "--organization", "org", "--stack", "stack", "ledger", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	assertDebugTraceContains(t, trace, "/membership/token", "/stack/api/auth/token", "/stack/api/ledger/v3/", "assertion", "[REDACTED]")
	f.assertNoSecrets(t, trace)
	out, trace = f.run(t, dir, "--debug", "--organization", "org", "--stack", "stack", "auth", "clients", "list")
	assertCLICloudJSON(t, out, `{"data":[]}`)
	assertDebugTraceContains(t, trace, "/stack/api/auth/clients", `request "GET"`, "response 200", "[REDACTED]")
	f.assertNoSecrets(t, trace)
}

func TestDebugServiceErrorKeepsStdoutEmpty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_info" {
			writeCLICloudJSON(t, w, map[string]string{"version": "3.0.0"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		writeCLICloudJSON(t, w, map[string]string{"errorCode": "UNAVAILABLE", "errorMessage": "debug-error-private"})
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	cacheDistributionLedger(t, dir, pluginmanager.Target{Endpoint: server.URL})
	out, trace, err := executeRoot(t, "--debug", "--config-dir", dir, "--auth-mode", "none", "--ledger-url", server.URL, "ledger", "list")
	if err == nil || out != "" || !strings.Contains(err.Error(), "UNAVAILABLE") {
		t.Fatalf("debug error changed CLI result: stdout=%q err=%v", out, err)
	}
	assertDebugTraceContains(t, trace, "response 503", "response body:", "[REDACTED]")
	assertDebugTraceHides(t, trace, "debug-error-private")
}

func TestDebugAuthSecretCreationRedactsOnlyTrace(t *testing.T) {
	t.Parallel()
	const clearValue = "auth-created-private-value"
	const payload = `{"data":{"id":"s1","name":"debug-secret","clear":"auth-created-private-value"}}`
	var calls atomic.Int32
	server := debugAuthSecretServer(t, payload, &calls)
	dir := t.TempDir()
	cacheDistributionAuth(t, dir, pluginmanager.Target{Endpoint: server.URL + "/prefix"})
	out, trace, err := executeRoot(t, "-d", "--config-dir", dir, "--auth-mode", "none", "--auth-url", server.URL+"/prefix", "auth", "clients", "secrets", "create", "c1", "--data", `{"name":"debug-secret"}`)
	if err != nil {
		t.Fatal(err)
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, []byte(payload), "", "  "); err != nil {
		t.Fatal(err)
	}
	if out != formatted.String()+"\n" || !strings.Contains(out, clearValue) {
		t.Fatal("stdout did not preserve the requested one-time clear secret")
	}
	assertDebugTraceContains(t, trace, `request "POST"`, "/prefix/clients/c1/secrets", "response 200", `"clear":"[REDACTED]"`)
	assertDebugTraceHides(t, trace, clearValue)
	if calls.Load() != 1 {
		t.Fatalf("debug repeated secret creation: %d calls", calls.Load())
	}
}

func debugAuthSecretServer(t *testing.T, payload string, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/prefix/_info" {
			writeCLICloudJSON(t, w, map[string]string{"version": "1.0.0"})
			return
		}
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/prefix/clients/c1/secrets" {
			t.Errorf("unexpected Auth request: %s %s", r.Method, r.URL.RequestURI())
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name != "debug-secret" {
			t.Errorf("secret creation body changed: name=%q err=%v", body.Name, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, payload); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func assertDebugTraceContains(t *testing.T, trace string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(trace, value) {
			t.Errorf("debug stderr missing %q: %s", value, trace)
		}
	}
}

func assertDebugTraceHides(t *testing.T, trace string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if strings.Contains(trace, secret) {
			t.Error("debug stderr exposed a fixture secret")
		}
	}
}
