package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v4config "github.com/formancehq/fctl/v4/internal/config"
	v4prompt "github.com/formancehq/fctl/v4/internal/prompt"
)

func executeCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	return executeCommandWithInput(t, "", args...)
}

func executeCommandWithInput(t *testing.T, input string, args ...string) (string, string, error) {
	t.Helper()

	command := NewRootCommand("test-version")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if input != "" {
		command.SetIn(strings.NewReader(input))
	}
	command.SetArgs(args)

	err := command.Execute()
	return stdout.String(), stderr.String(), err
}

func readRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return string(body)
}

func TestRootHelp(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "--help")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, expected := range []string{
		"Formance Control CLI v4",
		"--profile",
		"--organization",
		"--stack",
		"--config-dir",
		"-c, --config-dir",
		"-d, --debug",
		"--insecure-tls",
		"--no-color",
		"--non-interactive",
		"login",
		"logout",
		"profile",
		"version",
		"whoami",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected help output to contain %q, got:\n%s", expected, stdout)
		}
	}
	for _, hidden := range []string{"--context", "session", " context "} {
		if strings.Contains(stdout, hidden) {
			t.Fatalf("expected help output not to contain hidden %q, got:\n%s", hidden, stdout)
		}
	}
}

func TestPromptCancelledIsSilentExit(t *testing.T) {
	if !isSilentExitError(v4prompt.ErrCancelled) {
		t.Fatal("expected prompt cancellation to be a silent exit")
	}
	if !isSilentExitError(fmt.Errorf("wrapped: %w", v4prompt.ErrCancelled)) {
		t.Fatal("expected wrapped prompt cancellation to be a silent exit")
	}
	if isSilentExitError(errors.New("other error")) {
		t.Fatal("unexpected silent exit for unrelated error")
	}
}

func TestLoginOpenSourceWizardCreatesDefaultProfile(t *testing.T) {
	configDir := t.TempDir()

	stdout, stderr, err := executeCommandWithInput(t,
		"3\nhttp://localhost:8080\n",
		"--config-dir", configDir,
		"login",
	)
	if err != nil {
		t.Fatalf("login wizard open-source: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Logged in with profile default.") {
		t.Fatalf("unexpected login output:\n%s", stdout)
	}
	for _, expected := range []string{
		"Target\tFormance Open Source / local\n",
		"Stack URL\thttp://localhost:8080\n",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected login output to contain %q, got:\n%s", expected, stdout)
		}
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load login config: %v", err)
	}
	profile := cfg.Contexts["default"]
	if cfg.CurrentContext != "default" ||
		profile.Kind != v4config.ContextKindStack ||
		profile.StackURL != "http://localhost:8080" ||
		profile.Auth.Method != v4config.AuthMethodNone {
		t.Fatalf("unexpected default profile: current=%q profile=%#v", cfg.CurrentContext, profile)
	}
}

func TestLoginBrowserDeviceFlowCreatesCloudAndEEProfiles(t *testing.T) {
	for _, tc := range []struct {
		name       string
		target     string
		input      string
		wantKind   v4config.ContextKind
		extraFlags []string
		wantOutput []string
	}{
		{
			name:     "cloud",
			input:    "1\n1\n",
			wantKind: v4config.ContextKindCloud,
			wantOutput: []string{
				"Target\tFormance Cloud\n",
				"Authentication\tBrowser/device login\n",
			},
		},
		{
			name:     "ee",
			target:   "ee",
			input:    "1\n",
			wantKind: v4config.ContextKindCloudStack,
			extraFlags: []string{
				"--organization", "org_1",
				"--stack", "stack_1",
			},
			wantOutput: []string{
				"Authentication\tBrowser/device login\n",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configDir := t.TempDir()
			openedURL := ""
			previousOpenURL := loginOpenURL
			loginOpenURL = func(url string) error {
				openedURL = url
				return nil
			}
			t.Cleanup(func() {
				loginOpenURL = previousOpenURL
			})

			server := newDeviceLoginOIDCServer(t)
			defer server.Close()

			args := []string{"--config-dir", configDir}
			args = append(args, tc.extraFlags...)
			args = append(args,
				"login",
				"--membership-url", server.URL,
			)
			if tc.target != "" {
				args = append(args, "--target", tc.target)
			}

			stdout, stderr, err := executeCommandWithInput(t,
				tc.input,
				args...,
			)
			if err != nil {
				t.Fatalf("login browser/device: %v stderr=%s", err, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			if strings.Contains(stdout, "Static token") {
				t.Fatalf("login prompt must not offer static token auth, got:\n%s", stdout)
			}
			if !strings.Contains(stdout, "A browser window has been opened on https://verify.example?user_code=USER-CODE") ||
				!strings.Contains(stdout, "Waiting for authentication...") ||
				!strings.Contains(stdout, "Logged in with profile default.") {
				t.Fatalf("unexpected login output:\n%s", stdout)
			}
			for _, expected := range tc.wantOutput {
				if !strings.Contains(stdout, expected) {
					t.Fatalf("expected login output to contain %q, got:\n%s", expected, stdout)
				}
			}
			if openedURL != "https://verify.example?user_code=USER-CODE" {
				t.Fatalf("unexpected opened URL %q", openedURL)
			}

			cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
			if err != nil {
				t.Fatalf("load login config: %v", err)
			}
			profile := cfg.Contexts["default"]
			if cfg.CurrentContext != "default" ||
				profile.Kind != tc.wantKind ||
				profile.CloudURL != server.URL ||
				profile.Auth.Method != v4config.AuthMethodCloudDevice ||
				profile.Auth.IssuerURL != server.URL ||
				profile.Auth.TokenRef != "contexts/default/root-tokens" ||
				profile.Auth.Account != "user@example.com" {
				t.Fatalf("unexpected device login profile: current=%q profile=%#v", cfg.CurrentContext, profile)
			}
			if tc.wantKind == v4config.ContextKindCloudStack &&
				(profile.Organization != "org_1" || profile.Stack != "stack_1") {
				t.Fatalf("unexpected cloud-stack selection: %#v", profile)
			}
			storedTokens, err := os.ReadFile(filepath.Join(configDir, "credentials", "contexts", "default", "root-tokens"))
			if err != nil {
				t.Fatalf("expected stored root tokens: %v", err)
			}
			if !strings.Contains(string(storedTokens), "access-token") ||
				!strings.Contains(string(storedTokens), "refresh-token") {
				t.Fatalf("unexpected stored root tokens: %s", string(storedTokens))
			}
		})
	}
}

func newDeviceLoginOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_info":
			fmt.Fprint(w, `{"version":"v1.0.0","consoleURL":"https://portal.example"}`)
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"device_authorization_endpoint":%q,"token_endpoint":%q}`, server.URL+"/device", server.URL+"/token")
		case "/device":
			clientID, clientSecret, ok := r.BasicAuth()
			if !ok || clientID != "fctl" || clientSecret != "" {
				t.Fatalf("unexpected device basic auth: %q %q %v", clientID, clientSecret, ok)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse device form: %v", err)
			}
			if r.Form.Get("scope") != "openid offline_access accesses on_behalf" {
				t.Fatalf("unexpected device scope %q", r.Form.Get("scope"))
			}
			if r.Form.Get("prompt") != "no-org" {
				t.Fatalf("unexpected device prompt %q", r.Form.Get("prompt"))
			}
			fmt.Fprint(w, `{"device_code":"device-code","user_code":"USER-CODE","verification_uri":"https://verify.example","interval":1}`)
		case "/token":
			clientID, clientSecret, ok := r.BasicAuth()
			if !ok || clientID != "fctl" || clientSecret != "" {
				t.Fatalf("unexpected token basic auth: %q %q %v", clientID, clientSecret, ok)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" ||
				r.Form.Get("device_code") != "device-code" {
				t.Fatalf("unexpected token form: %s", r.Form.Encode())
			}
			fmt.Fprintf(w, `{"access_token":"access-token","token_type":"Bearer","refresh_token":"refresh-token","id_token":%q,"expires_in":3600}`,
				testJWT(t, map[string]any{"email": "user@example.com"}))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	return server
}

func testJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "none"})
	if err != nil {
		t.Fatalf("marshal jwt header: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal jwt payload: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
}

func TestLoginDoesNotExposeStaticTokenAuth(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "login", "--help")
	if err != nil {
		t.Fatalf("login help: %v stderr=%s", err, stderr)
	}
	for _, hidden := range []string{"Static token", "--token", "--token-stdin"} {
		if strings.Contains(stdout, hidden) {
			t.Fatalf("login help must not expose %q, got:\n%s", hidden, stdout)
		}
	}

	_, stderr, err = executeCommand(t, "login", "--target", "cloud", "--token", "secret")
	if err == nil {
		t.Fatal("expected --token to be rejected by login")
	}
	if !strings.Contains(err.Error(), "unknown flag: --token") {
		t.Fatalf("unexpected --token error: %v stderr=%s", err, stderr)
	}
}

func TestLoginCloudClientCredentialsAndLogout(t *testing.T) {
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"--profile", "production",
		"--organization", "org_1",
		"--stack", "stack_1",
		"login",
		"--target", "cloud",
		"--client-id", "client",
		"--client-secret", "super-secret",
	)
	if err != nil {
		t.Fatalf("login cloud client credentials: %v stderr=%s", err, stderr)
	}
	if stdout != "Logged in with profile production.\n" {
		t.Fatalf("unexpected login output: %q", stdout)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load login config: %v", err)
	}
	profile := cfg.Contexts["production"]
	if cfg.CurrentContext != "production" ||
		profile.Kind != v4config.ContextKindCloudStack ||
		profile.CloudURL != v4config.DefaultCloudURL ||
		profile.Organization != "org_1" ||
		profile.Stack != "stack_1" ||
		profile.Auth.Method != v4config.AuthMethodClientCredentials ||
		profile.Auth.ClientID != "client" ||
		profile.Auth.SecretRef != "contexts/production/client-secret" {
		t.Fatalf("unexpected production profile: current=%q profile=%#v", cfg.CurrentContext, profile)
	}
	if !stringSliceContains(profile.Auth.Scopes, "organization:ListStacks") {
		t.Fatalf("expected cloud client credentials to include organization scopes, got %#v", profile.Auth.Scopes)
	}
	if strings.Contains(fmt.Sprintf("%#v", profile), "super-secret") {
		t.Fatalf("profile must not contain clear client secret: %#v", profile)
	}
	if _, err := os.Stat(filepath.Join(configDir, "credentials", "contexts", "production", "client-secret")); err != nil {
		t.Fatalf("expected stored client secret: %v", err)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"whoami",
	)
	if err != nil {
		t.Fatalf("whoami: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Profile\tproduction\n",
		"Target\tcloud-stack\n",
		"Auth\tclient_credentials\n",
		"Organization\torg_1\n",
		"Stack\tstack_1\n",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected whoami output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"logout",
	)
	if err != nil {
		t.Fatalf("logout: %v stderr=%s", err, stderr)
	}
	if stdout != "Logged out from profile production.\n" {
		t.Fatalf("unexpected logout output: %q", stdout)
	}
	cfg, err = v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load logout config: %v", err)
	}
	if cfg.Contexts["production"].Auth.Method != v4config.AuthMethodNone {
		t.Fatalf("expected logout to clear auth: %#v", cfg.Contexts["production"].Auth)
	}
}

func TestLoginNonInteractiveDoesNotPrompt(t *testing.T) {
	configDir := t.TempDir()

	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"--non-interactive",
		"login",
	)
	if err == nil {
		t.Fatal("expected missing target to fail in non-interactive mode")
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "login requires --target in non-interactive mode") {
		t.Fatalf("unexpected missing target error: %v", err)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--non-interactive",
		"login",
		"--target", "open-source",
	)
	if err == nil {
		t.Fatal("expected missing stack URL to fail in non-interactive mode")
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "login open-source requires --stack-url") {
		t.Fatalf("unexpected missing stack URL error: %v", err)
	}
}

func TestInsecureTLSFlagAllowsSelfSignedStackTarget(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "tls-local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create TLS context: %v stderr=%s", err, stderr)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "target", "inspect")
	if err == nil {
		t.Fatal("expected self-signed TLS target to fail without --insecure-tls")
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "--insecure-tls", "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target with --insecure-tls: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Context: tls-local") || !strings.Contains(stdout, "ledger 2.3.4 healthy") {
		t.Fatalf("unexpected inspect output:\n%s", stdout)
	}
}

func TestConfigDirShortFlag(t *testing.T) {
	configDir := t.TempDir()
	stdout, stderr, err := executeCommand(t,
		"-c", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
	)
	if err != nil {
		t.Fatalf("create context with -c: %v stderr=%s", err, stderr)
	}
	if stdout != "Context local created.\n" {
		t.Fatalf("unexpected create output: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(configDir, "config.yaml")); err != nil {
		t.Fatalf("expected config written through -c: %v", err)
	}
}

func TestVersionCommand(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "version")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout != "fctl v4 test-version\n" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
}

func TestSetupAndPromptAlias(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "setup")
	if err != nil {
		t.Fatalf("setup: %v stderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "fctl login") ||
		!strings.Contains(stdout, "fctl profile create stack <name> --stack-url <url>") {
		t.Fatalf("unexpected setup output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "prompt")
	if err != nil {
		t.Fatalf("prompt alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command prompt has been deprecated, use setup or login") {
		t.Fatalf("expected prompt warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "fctl login") ||
		!strings.Contains(stdout, "fctl profile create stack <name> --stack-url <url>") {
		t.Fatalf("unexpected prompt output:\n%s", stdout)
	}
}

func TestUIPrintsCloudConsoleURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_info" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"version":"test","consoleURL":"https://console.example"}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "ui", "--print")
	if err != nil {
		t.Fatalf("cloud ui print: %v stderr=%s", err, stderr)
	}
	if stdout != "Console URL: https://console.example\n" {
		t.Fatalf("unexpected cloud ui output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "ui", "--print")
	if err != nil {
		t.Fatalf("root ui alias print: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command ui has been deprecated, use cloud ui") {
		t.Fatalf("expected root ui deprecation warning, got:\n%s", stderr)
	}
	if stdout != "Console URL: https://console.example\n" {
		t.Fatalf("unexpected root ui alias output: %q", stdout)
	}
}

func TestUIRejectsStackContext(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
	)
	if err != nil {
		t.Fatalf("create stack context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "ui", "--print")
	if err == nil {
		t.Fatal("expected cloud ui to reject stack contexts")
	}
	if !strings.Contains(err.Error(), "cloud commands require a cloud or cloud-stack context") {
		t.Fatalf("unexpected ui error: %v stderr=%s", err, stderr)
	}
}

func TestContextCreateListShowUse(t *testing.T) {
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context local created.\n" {
		t.Fatalf("unexpected create output: %q", stdout)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.CurrentContext != "local" {
		t.Fatalf("expected current context local, got %q", cfg.CurrentContext)
	}
	if cfg.Contexts["local"].Defaults["ledger"] != "default" {
		t.Fatalf("expected default ledger to be persisted")
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "list")
	if err != nil {
		t.Fatalf("list contexts: %v stderr=%s", err, stderr)
	}
	if stdout != "* local\n" {
		t.Fatalf("unexpected list output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "show", "local")
	if err != nil {
		t.Fatalf("show context: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Name: local") || !strings.Contains(stdout, "Kind: stack") {
		t.Fatalf("unexpected show output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "other",
		"--stack-url", "http://other/api",
	)
	if err != nil {
		t.Fatalf("create second context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context other created.\n" {
		t.Fatalf("unexpected second create output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "use", "other")
	if err != nil {
		t.Fatalf("use context: %v stderr=%s", err, stderr)
	}
	if stdout != "Current context set to other.\n" {
		t.Fatalf("unexpected use output: %q", stdout)
	}

	cfg, err = v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config after use: %v", err)
	}
	if cfg.CurrentContext != "other" {
		t.Fatalf("expected current context other, got %q", cfg.CurrentContext)
	}
}

func TestContextCreateCloudAndCloudStack(t *testing.T) {
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", "https://cloud.example/api",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context cloud created.\n" {
		t.Fatalf("unexpected cloud create output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", "https://cloud.example/api",
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context prod created.\n" {
		t.Fatalf("unexpected cloud-stack create output: %q", stdout)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Contexts["cloud"].Kind != v4config.ContextKindCloud {
		t.Fatalf("expected cloud context kind, got %q", cfg.Contexts["cloud"].Kind)
	}
	prod := cfg.Contexts["prod"]
	if prod.Kind != v4config.ContextKindCloudStack || prod.Organization != "org_1" || prod.Stack != "stack_1" {
		t.Fatalf("unexpected cloud-stack context: %#v", prod)
	}
	if prod.Defaults["ledger"] != "default" {
		t.Fatalf("expected default ledger")
	}
}

func TestGlobalOrganizationStackOverride(t *testing.T) {
	stackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"3.2.4","health":true}]}`)
	}))
	defer stackServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/org_1/stacks/stack_1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":%q,"regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}}`, stackServer.URL)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"profile", "create", "cloud", "cloud",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud profile: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"--organization", "org_1",
		"--stack", "stack_1",
		"target", "inspect",
	)
	if err != nil {
		t.Fatalf("inspect cloud-stack override: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Target: "+stackServer.URL+" (cloud-stack)") ||
		!strings.Contains(stdout, "ledger 3.2.4 healthy") {
		t.Fatalf("unexpected target inspect output:\n%s", stdout)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--stack", "stack_1",
		"target", "inspect",
	)
	if err == nil {
		t.Fatal("expected stack override without organization to fail")
	}
	if !strings.Contains(err.Error(), "organization is required for cloud-stack targets") {
		t.Fatalf("unexpected missing organization error: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"profile", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create stack profile: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--profile", "local",
		"--organization", "org_1",
		"target", "inspect",
	)
	if err == nil {
		t.Fatal("expected organization override on stack profile to fail")
	}
	if !strings.Contains(err.Error(), "--organization and --stack can only be used with Cloud or EE profiles") {
		t.Fatalf("unexpected stack profile override error: %v stderr=%s", err, stderr)
	}
}

func TestContextSetUpdatesCloudStack(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", "https://cloud.example/api",
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "set", "prod",
		"--organization", "org_2",
		"--stack", "stack_2",
		"--default-ledger", "ledger_2",
	)
	if err != nil {
		t.Fatalf("set context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context prod updated.\n" {
		t.Fatalf("unexpected set output: %q", stdout)
	}
	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	prod := cfg.Contexts["prod"]
	if prod.Organization != "org_2" || prod.Stack != "stack_2" || prod.Defaults["ledger"] != "ledger_2" {
		t.Fatalf("unexpected updated context: %#v", prod)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "context", "unset-defaults", "prod")
	if err == nil {
		t.Fatal("expected context unset-defaults to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "unset-defaults", "prod", "--confirm")
	if err != nil {
		t.Fatalf("unset context defaults: %v stderr=%s", err, stderr)
	}
	if stdout != "Context prod defaults cleared.\n" {
		t.Fatalf("unexpected unset-defaults output: %q", stdout)
	}
	cfg, err = v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config after unset defaults: %v", err)
	}
	if cfg.Contexts["prod"].Defaults != nil {
		t.Fatalf("expected defaults to be cleared, got %#v", cfg.Contexts["prod"].Defaults)
	}
}

func TestProfilesDefaultAliases(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", "https://cloud.example/api",
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "profiles", "set-default-organization", "org_2")
	if err != nil {
		t.Fatalf("profiles set-default-organization: %v stderr=%s", err, stderr)
	}
	if stdout != "Context prod updated.\n" {
		t.Fatalf("unexpected set-default-organization output: %q", stdout)
	}
	if !strings.Contains(stderr, "Command profiles has been deprecated, use profile") ||
		!strings.Contains(stderr, "use profile set --organization <organization-id>") {
		t.Fatalf("expected profiles organization deprecation warnings, got:\n%s", stderr)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "profiles", "set-default-stack", "stack_2")
	if err != nil {
		t.Fatalf("profiles set-default-stack: %v stderr=%s", err, stderr)
	}
	if stdout != "Context prod updated.\n" {
		t.Fatalf("unexpected set-default-stack output: %q", stdout)
	}
	if !strings.Contains(stderr, "Command profiles has been deprecated, use profile") ||
		!strings.Contains(stderr, "use profile set --stack <stack-id>") {
		t.Fatalf("expected profiles stack deprecation warnings, got:\n%s", stderr)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Contexts["prod"].Organization != "org_2" || cfg.Contexts["prod"].Stack != "stack_2" {
		t.Fatalf("expected profile aliases to update cloud-stack defaults: %#v", cfg.Contexts["prod"])
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "profiles", "reset", "prod", "--confirm")
	if err != nil {
		t.Fatalf("profiles reset: %v stderr=%s", err, stderr)
	}
	if stdout != "Context prod defaults cleared.\n" {
		t.Fatalf("unexpected profiles reset output: %q", stdout)
	}
	if !strings.Contains(stderr, "Command profiles has been deprecated, use profile") ||
		!strings.Contains(stderr, "use profile unset-defaults <name> --confirm") {
		t.Fatalf("expected profiles reset deprecation warnings, got:\n%s", stderr)
	}
	cfg, err = v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config after reset: %v", err)
	}
	if cfg.Contexts["prod"].Defaults != nil {
		t.Fatalf("expected profile reset alias to clear defaults, got %#v", cfg.Contexts["prod"].Defaults)
	}
}

func TestContextRenameAndDelete(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
	)
	if err != nil {
		t.Fatalf("create local context: %v stderr=%s", err, stderr)
	}
	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "other",
		"--stack-url", "http://other/api",
	)
	if err != nil {
		t.Fatalf("create other context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "context", "rename", "local", "renamed")
	if err != nil {
		t.Fatalf("rename context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context local renamed to renamed.\n" {
		t.Fatalf("unexpected rename output: %q", stdout)
	}
	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.CurrentContext != "renamed" {
		t.Fatalf("expected current context renamed, got %q", cfg.CurrentContext)
	}
	if _, ok := cfg.Contexts["local"]; ok {
		t.Fatalf("old context name still exists")
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "context", "delete", "renamed")
	if err == nil {
		t.Fatal("expected context delete to require --confirm")
	}
	_, _, err = executeCommand(t, "--config-dir", configDir, "context", "delete", "renamed", "--confirm")
	if err == nil {
		t.Fatal("expected current context delete to require --force")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "context", "delete", "renamed", "--confirm", "--force")
	if err != nil {
		t.Fatalf("delete context: %v stderr=%s", err, stderr)
	}
	if stdout != "Context renamed deleted.\n" {
		t.Fatalf("unexpected delete output: %q", stdout)
	}
	cfg, err = v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load config after delete: %v", err)
	}
	if _, ok := cfg.Contexts["renamed"]; ok {
		t.Fatalf("deleted context still exists")
	}
	if cfg.CurrentContext != "" {
		t.Fatalf("expected current context to be cleared, got %q", cfg.CurrentContext)
	}
}

func TestProfilesDeprecatedAlias(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "profiles", "list")
	if err != nil {
		t.Fatalf("profiles list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command profiles has been deprecated, use profile") {
		t.Fatalf("expected profiles warning, got:\n%s", stderr)
	}
	if stdout != "* local\n" {
		t.Fatalf("unexpected profiles list output: %q", stdout)
	}
}

func TestCloudMeShow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":"user_1","email":"user@example.com","role":"USER"}}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "me", "show")
	if err != nil {
		t.Fatalf("cloud me show: %v stderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, expected := range []string{"ID\tuser_1", "Email\tuser@example.com", "Role\tUSER"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected cloud me output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestCloudMeInvitations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/me/invitations":
			if got := r.URL.Query().Get("status"); got != "PENDING" {
				t.Fatalf("expected status PENDING, got %q", got)
			}
			if got := r.URL.Query().Get("organization"); got != "org_1" {
				t.Fatalf("expected organization org_1, got %q", got)
			}
			fmt.Fprint(w, `{"data":[{"id":"inv_1","organizationId":"org_1","userEmail":"user@example.com","status":"PENDING","creationDate":"2026-01-01T00:00:00Z"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/me/invitations/inv_1/accept":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/me/invitations/inv_1/reject":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "me", "invitations", "list", "--status", "PENDING", "--organization", "org_1")
	if err != nil {
		t.Fatalf("cloud me invitations list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "inv_1\torg_1\tuser@example.com\tPENDING") {
		t.Fatalf("unexpected invitations list output:\n%s", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "me", "invitations", "accept", "inv_1")
	if err == nil {
		t.Fatal("expected invitation accept to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "me", "invitations", "accept", "inv_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud me invitations accept: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud invitation inv_1 accepted.\n" {
		t.Fatalf("unexpected accept output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "me", "invitations", "decline", "inv_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud me invitations decline: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud invitation inv_1 declined.\n" {
		t.Fatalf("unexpected decline output: %q", stdout)
	}
}

func TestCloudOrganizationsListAndShow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/organizations":
			fmt.Fprint(w, `{"data":[{"id":"org_1","name":"Acme","ownerId":"user_1","domain":"acme.test","totalStacks":2,"totalUsers":3}]}`)
		case "/organizations/org_1":
			fmt.Fprint(w, `{"data":{"id":"org_1","name":"Acme","ownerId":"user_1","domain":"acme.test","totalStacks":2,"totalUsers":3}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "list")
	if err != nil {
		t.Fatalf("cloud organizations list: %v stderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "org_1\tAcme\tuser_1") {
		t.Fatalf("unexpected organizations list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "describe", "org_1")
	if err != nil {
		t.Fatalf("cloud organizations describe: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command cloud organizations describe has been deprecated, use cloud organizations show") {
		t.Fatalf("expected describe deprecation warning, got:\n%s", stderr)
	}
	for _, expected := range []string{"ID\torg_1", "Name\tAcme", "Owner\tuser_1", "Domain\tacme.test"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected organization output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestCloudDeviceUsesRootTokenForMembershipAndOrganizationTokenForStacks(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/_info":
			fmt.Fprint(w, `{"version":"v1.0.0","consoleURL":"https://portal.example"}`)
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"device_authorization_endpoint":%q,"token_endpoint":%q}`, server.URL+"/device", server.URL+"/token")
		case "/organizations":
			if got := r.Header.Get("Authorization"); got != "Bearer root-token" {
				t.Fatalf("expected root token for organizations list, got %q", got)
			}
			fmt.Fprint(w, `{"data":[{"id":"org_1","name":"Acme","ownerId":"user_1","domain":"acme.test","totalStacks":1,"totalUsers":3}]}`)
		case "/device":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse device form: %v", err)
			}
			if r.Form.Get("organization_id") != "org_1" {
				t.Fatalf("unexpected organization_id %q", r.Form.Get("organization_id"))
			}
			if r.Form.Get("id_token_hint") != "root-id-token" {
				t.Fatalf("unexpected id_token_hint %q", r.Form.Get("id_token_hint"))
			}
			if !strings.Contains(r.Form.Get("scope"), "openid offline_access") ||
				!strings.Contains(r.Form.Get("scope"), "organization:ListStacks") {
				t.Fatalf("unexpected organization device scope %q", r.Form.Get("scope"))
			}
			fmt.Fprint(w, `{"device_code":"org-device-code","user_code":"ORG-CODE","verification_uri":"https://verify.example","interval":1}`)
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" ||
				r.Form.Get("device_code") != "org-device-code" {
				t.Fatalf("unexpected token form: %s", r.Form.Encode())
			}
			fmt.Fprint(w, `{"access_token":"org-token","token_type":"Bearer","refresh_token":"org-refresh-token","expires_in":3600}`)
		case "/organizations/org_1/stacks":
			if got := r.Header.Get("Authorization"); got != "Bearer org-token" {
				t.Fatalf("expected organization token for stacks list, got %q", got)
			}
			fmt.Fprint(w, `{"data":[{"id":"stack_1","name":"Production","organizationId":"org_1","uri":"https://stack.example/api","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"auditEnabled":false,"synchronised":true,"modules":[]}]}`)
		case "/organizations/org_1/stacks/stack_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("unexpected method %s for stack endpoint", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer org-token" {
				t.Fatalf("expected organization token for stack delete, got %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	openedURL := ""
	previousOpenURL := loginOpenURL
	loginOpenURL = func(url string) error {
		openedURL = url
		return nil
	}
	t.Cleanup(func() {
		loginOpenURL = previousOpenURL
	})

	configDir := t.TempDir()
	cfg := v4config.Config{
		Version:        v4config.Version,
		CurrentContext: "cloud",
		Contexts: map[string]v4config.Context{
			"cloud": {
				Kind:     v4config.ContextKindCloud,
				CloudURL: server.URL,
				Auth: v4config.Auth{
					Method:    v4config.AuthMethodCloudDevice,
					IssuerURL: server.URL,
					TokenRef:  "contexts/cloud/root-tokens",
					Scopes:    []string{"organization:ListStacks"},
				},
			},
		},
	}
	if err := v4config.SaveFile(filepath.Join(configDir, "config.yaml"), cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tokenPath := filepath.Join(configDir, "credentials", "contexts", "cloud", "root-tokens")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatalf("create credential dir: %v", err)
	}
	if err := os.WriteFile(tokenPath, []byte(`{"accessToken":{"token":"root-token","tokenType":"Bearer","refreshToken":"root-refresh"},"idToken":"root-id-token","refreshToken":"root-refresh"}`), 0o600); err != nil {
		t.Fatalf("write root token: %v", err)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "list")
	if err != nil {
		t.Fatalf("cloud organizations list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "org_1\tAcme\tuser_1") {
		t.Fatalf("unexpected organizations output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "list", "--organization", "org_1")
	if err != nil {
		t.Fatalf("cloud stacks list: %v stderr=%s", err, stderr)
	}
	if openedURL != "https://verify.example?user_code=ORG-CODE" {
		t.Fatalf("unexpected organization auth URL %q", openedURL)
	}
	if !strings.Contains(stdout, "stack_1") {
		t.Fatalf("unexpected stacks output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "delete", "stack_1", "--organization", "org_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud stacks delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 deleted.\n" {
		t.Fatalf("unexpected delete output: %q", stdout)
	}
}

func TestLedgerInfoCloudDeviceScopesOrganizationAndStackTokens(t *testing.T) {
	var stackServer *httptest.Server
	stackServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"token_endpoint":%q}`, stackServer.URL+"/api/auth/token")
		case "/api/auth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse stack token form: %v", err)
			}
			if r.Form.Get("assertion") != "stack-assertion" {
				t.Fatalf("unexpected stack assertion %q", r.Form.Get("assertion"))
			}
			fmt.Fprint(w, `{"access_token":"stack-token","token_type":"Bearer","expires_in":3600}`)
		case "/versions":
			if got := r.Header.Get("Authorization"); got != "Bearer stack-token" {
				t.Fatalf("expected stack token on versions, got %q", got)
			}
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"1.9.0","health":true}]}`)
		case "/api/ledger/_info":
			if got := r.Header.Get("Authorization"); got != "Bearer stack-token" {
				t.Fatalf("expected stack token on ledger info, got %q", got)
			}
			fmt.Fprint(w, `{"data":{"server":"ledger","version":"1.9.0","config":{}}}`)
		default:
			t.Fatalf("unexpected stack path %s", r.URL.Path)
		}
	}))
	defer stackServer.Close()

	var membershipServer *httptest.Server
	membershipServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"device_authorization_endpoint":%q,"token_endpoint":%q}`, membershipServer.URL+"/device", membershipServer.URL+"/token")
		case "/device":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse device form: %v", err)
			}
			if r.Form.Get("organization_id") != "org_1" {
				t.Fatalf("unexpected organization_id %q", r.Form.Get("organization_id"))
			}
			if r.Form.Get("id_token_hint") != "root-id-token" {
				t.Fatalf("unexpected id_token_hint %q", r.Form.Get("id_token_hint"))
			}
			switch {
			case r.Form.Get("resource") == "":
				if !strings.Contains(r.Form.Get("scope"), "organization:ReadStack") {
					t.Fatalf("expected organization scopes, got %q", r.Form.Get("scope"))
				}
				fmt.Fprint(w, `{"device_code":"org-device-code","user_code":"ORG-CODE","verification_uri":"https://verify.example","interval":1}`)
			case r.Form.Get("resource") == "stack://org_1/stack_1|stack:Read stack:Write":
				if r.Form.Get("scope") != "openid offline_access" {
					t.Fatalf("unexpected stack scope %q", r.Form.Get("scope"))
				}
				fmt.Fprint(w, `{"device_code":"stack-device-code","user_code":"STACK-CODE","verification_uri":"https://verify.example","interval":1}`)
			default:
				t.Fatalf("unexpected resource %q", r.Form.Get("resource"))
			}
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			switch r.Form.Get("device_code") {
			case "org-device-code":
				fmt.Fprint(w, `{"access_token":"org-token","token_type":"Bearer","refresh_token":"org-refresh-token","expires_in":3600}`)
			case "stack-device-code":
				if r.Form.Get("resource") != "stack://org_1/stack_1|stack:Read stack:Write" {
					t.Fatalf("unexpected token resource %q", r.Form.Get("resource"))
				}
				fmt.Fprint(w, `{"access_token":"stack-assertion","token_type":"Bearer","refresh_token":"stack-refresh-token","expires_in":3600}`)
			default:
				t.Fatalf("unexpected device_code %q", r.Form.Get("device_code"))
			}
		case "/organizations/org_1/stacks/stack_1":
			if got := r.Header.Get("Authorization"); got != "Bearer org-token" {
				t.Fatalf("expected organization token on stack lookup, got %q", got)
			}
			fmt.Fprintf(w, `{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":%q,"regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}}`, stackServer.URL)
		default:
			t.Fatalf("unexpected membership path %s", r.URL.Path)
		}
	}))
	defer membershipServer.Close()

	openedURLs := []string{}
	previousOpenURL := loginOpenURL
	loginOpenURL = func(url string) error {
		openedURLs = append(openedURLs, url)
		return nil
	}
	t.Cleanup(func() {
		loginOpenURL = previousOpenURL
	})

	configDir := t.TempDir()
	cfg := v4config.Config{
		Version:        v4config.Version,
		CurrentContext: "cloud",
		Contexts: map[string]v4config.Context{
			"cloud": {
				Kind:     v4config.ContextKindCloud,
				CloudURL: membershipServer.URL,
				Auth: v4config.Auth{
					Method:    v4config.AuthMethodCloudDevice,
					IssuerURL: membershipServer.URL,
					TokenRef:  "contexts/cloud/root-tokens",
				},
			},
		},
	}
	if err := v4config.SaveFile(filepath.Join(configDir, "config.yaml"), cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
	tokenPath := filepath.Join(configDir, "credentials", "contexts", "cloud", "root-tokens")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatalf("create credential dir: %v", err)
	}
	if err := os.WriteFile(tokenPath, []byte(`{"accessToken":{"token":"root-token","tokenType":"Bearer","refreshToken":"root-refresh"},"idToken":"root-id-token","refreshToken":"root-refresh"}`), 0o600); err != nil {
		t.Fatalf("write root token: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"--organization", "org_1",
		"--stack", "stack_1",
		"ledger", "info",
	)
	if err != nil {
		t.Fatalf("ledger info cloud device: %v stderr=%s", err, stderr)
	}
	if len(openedURLs) != 2 ||
		openedURLs[0] != "https://verify.example?user_code=ORG-CODE" ||
		openedURLs[1] != "https://verify.example?user_code=STACK-CODE" {
		t.Fatalf("unexpected opened URLs: %#v", openedURLs)
	}
	for _, expected := range []string{"API version: v1", "Server", "ledger", "Version", "1.9.0"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger info output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestCloudOrganizationsHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/organizations/org_1/logs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if got := r.URL.Query().Get("pageSize"); got != "10" {
			t.Fatalf("expected pageSize 10, got %q", got)
		}
		if got := r.URL.Query().Get("action"); got != "regions.created" {
			t.Fatalf("expected action regions.created, got %q", got)
		}
		if got := r.URL.Query().Get("userId"); got != "user_1" {
			t.Fatalf("expected userId user_1, got %q", got)
		}
		if got := r.URL.Query().Get("key"); got != "region.id" {
			t.Fatalf("expected key region.id, got %q", got)
		}
		if got := r.URL.Query().Get("value"); got != "reg_1" {
			t.Fatalf("expected value reg_1, got %q", got)
		}
		fmt.Fprint(w, `{"data":{"pageSize":10,"hasMore":false,"data":[{"seq":"1","organizationId":"org_1","userId":"user_1","action":"regions.created","date":"2026-01-01T00:00:00Z","data":{}}]}}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "history", "--action", "regions.created", "--user-id", "user_1", "--data", "region.id=reg_1")
	if err != nil {
		t.Fatalf("cloud organizations history: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "1\torg_1\tuser_1\tregions.created\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected history output:\n%s", stdout)
	}
}

func TestCloudOrganizationsMutations(t *testing.T) {
	orgBody := `{"data":{"id":"org_1","name":"Acme","ownerId":"user_1","domain":"acme.test","defaultPolicyID":42}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/organizations":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Acme"`, `"domain":"acme.test"`, `"defaultPolicyID":42`, `"ownerID":"user_1"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create organization body to contain %s, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, orgBody)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Acme Updated"`, `"domain":"acme.test"`, `"defaultPolicyID":42`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected update organization body to contain %s, got %s", expected, body)
				}
			}
			fmt.Fprint(w, orgBody)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"cloud", "organizations", "create", "Acme",
		"--domain", "acme.test",
		"--default-policy-id", "42",
		"--owner-id", "user_1",
	)
	if err != nil {
		t.Fatalf("cloud organizations create: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud organization org_1 created.\n" {
		t.Fatalf("unexpected create output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"cloud", "organizations", "update", "org_1",
		"--name", "Acme Updated",
		"--domain", "acme.test",
		"--default-policy-id", "42",
	)
	if err != nil {
		t.Fatalf("cloud organizations update: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud organization org_1 updated.\n" {
		t.Fatalf("unexpected update output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "delete", "org_1")
	if err == nil {
		t.Fatal("expected organization delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "delete", "org_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud organization org_1 deleted.\n" {
		t.Fatalf("unexpected delete output: %q", stdout)
	}
}

func TestCloudOrganizationApplications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/applications":
			if got := r.URL.Query().Get("page"); got != "0" {
				t.Fatalf("expected page 0, got %q", got)
			}
			if got := r.URL.Query().Get("pageSize"); got != "15" {
				t.Fatalf("expected pageSize 15, got %q", got)
			}
			fmt.Fprint(w, `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"id":"app_1","name":"Console","alias":"console","url":"https://console.example","description":"Cloud console","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/applications/app_1":
			fmt.Fprint(w, `{"data":{"id":"app_1","name":"Console","alias":"console","url":"https://console.example","description":"Cloud console","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","scopes":[{"id":7,"label":"ledger:read","protected":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}]}}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "applications", "list")
	if err != nil {
		t.Fatalf("cloud organizations applications list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "app_1\tConsole\tconsole\thttps://console.example") {
		t.Fatalf("unexpected applications list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "applications", "show", "app_1")
	if err != nil {
		t.Fatalf("cloud organizations applications show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ID\tapp_1", "Name\tConsole", "Alias\tconsole", "URL\thttps://console.example", "Description\tCloud console", "Scope\t7\tledger:read"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected application output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestCloudOrganizationAuthenticationProvider(t *testing.T) {
	providerBody := `{"data":{"type":"oidc","name":"Acme OIDC","clientID":"client_1","clientSecret":"secret","config":{"issuer":"https://issuer.example"},"organizationID":"org_1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","redirectURI":"https://cloud.example/callback"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/authentication-provider":
			fmt.Fprint(w, providerBody)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/authentication-provider":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"type":"oidc"`, `"name":"Acme OIDC"`, `"clientID":"client_1"`, `"clientSecret":"secret"`, `"issuer":"https://issuer.example"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected authentication provider body to contain %s, got %s", expected, body)
				}
			}
			fmt.Fprint(w, providerBody)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/authentication-provider":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "authentication-provider", "show")
	if err != nil {
		t.Fatalf("cloud organizations authentication-provider show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"Type\toidc", "Name\tAcme OIDC", "ClientID\tclient_1", "RedirectURI\thttps://cloud.example/callback", "Issuer\thttps://issuer.example"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected provider output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"cloud", "organizations", "authentication-provider", "configure",
		"--type", "oidc",
		"--name", "Acme OIDC",
		"--client-id", "client_1",
		"--client-secret", "secret",
		"--oidc-issuer", "https://issuer.example",
	)
	if err != nil {
		t.Fatalf("cloud organizations authentication-provider configure: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud authentication provider Acme OIDC configured.\n" {
		t.Fatalf("unexpected configure output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "authentication-provider", "delete")
	if err == nil {
		t.Fatal("expected authentication provider delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "authentication-provider", "delete", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations authentication-provider delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud authentication provider for organization org_1 deleted.\n" {
		t.Fatalf("unexpected delete output: %q", stdout)
	}
}

func TestCloudOrganizationAuthenticationProviderConfigureDeprecatedPositionalArgs(t *testing.T) {
	providerBody := `{"data":{"type":"github","name":"GitHub","clientID":"client_1","clientSecret":"secret","config":{},"organizationID":"org_1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/authentication-provider":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"type":"github"`, `"name":"GitHub"`, `"clientID":"client_1"`, `"clientSecret":"secret"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected authentication provider body to contain %s, got %s", expected, body)
				}
			}
			fmt.Fprint(w, providerBody)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"cloud", "organizations", "authentication-provider", "configure",
		"github", "GitHub", "client_1", "secret",
	)
	if err != nil {
		t.Fatalf("cloud organizations authentication-provider configure positional: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Positional authentication provider arguments have been deprecated") {
		t.Fatalf("expected positional deprecation warning, got:\n%s", stderr)
	}
	if stdout != "Cloud authentication provider GitHub configured.\n" {
		t.Fatalf("unexpected configure output: %q", stdout)
	}
}

func TestCloudOrganizationOAuthClients(t *testing.T) {
	clientBody := `{"data":{"id":"client_1","name":"Robot","description":"CI client","secret":{"lastDigits":"1234"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`
	createdClientBody := `{"data":{"id":"client_1","name":"Robot","description":"CI client","secret":{"lastDigits":"1234","clear":"clear-secret"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/clients":
			fmt.Fprint(w, `{"data":{"pageSize":15,"hasMore":false,"data":[{"id":"client_1","name":"Robot","description":"CI client","secret":{"lastDigits":"1234"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/clients/client_1":
			fmt.Fprint(w, clientBody)
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/clients":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Robot"`, `"description":"CI client"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected OAuth client create body to contain %s, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, createdClientBody)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/clients/client_1":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Robot Updated"`, `"description":"Updated client"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected OAuth client update body to contain %s, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/clients/client_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "list")
	if err != nil {
		t.Fatalf("cloud organizations oauth-clients list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "organization_client_1\tRobot\t1234\tCI client") {
		t.Fatalf("unexpected oauth clients list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "show", "organization_client_1")
	if err != nil {
		t.Fatalf("cloud organizations oauth-clients show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ClientID\torganization_client_1", "Name\tRobot", "SecretLastDigits\t1234", "Description\tCI client"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected oauth client output to contain %q, got:\n%s", expected, stdout)
		}
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "create", "--name", "Robot", "--description", "CI client")
	if err == nil {
		t.Fatal("expected oauth client create to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "create", "--name", "Robot", "--description", "CI client", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations oauth-clients create: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ClientID\torganization_client_1", "Secret\tclear-secret"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected create output to contain %q, got:\n%s", expected, stdout)
		}
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "update", "organization_client_1", "--name", "Robot Updated", "--description", "Updated client")
	if err == nil {
		t.Fatal("expected oauth client update to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "update", "organization_client_1", "--name", "Robot Updated", "--description", "Updated client", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations oauth-clients update: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud OAuth client organization_client_1 updated.\n" {
		t.Fatalf("unexpected update output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "delete", "organization_client_1")
	if err == nil {
		t.Fatal("expected oauth client delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "oauth-clients", "delete", "organization_client_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations oauth-clients delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud OAuth client organization_client_1 deleted.\n" {
		t.Fatalf("unexpected delete output: %q", stdout)
	}
}

func TestCloudOrganizationInvitations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/invitations":
			if got := r.URL.Query().Get("status"); got != "PENDING" {
				t.Fatalf("expected status PENDING, got %q", got)
			}
			fmt.Fprint(w, `{"data":[{"id":"inv_1","organizationId":"org_1","userEmail":"user@example.com","status":"PENDING","creationDate":"2026-01-01T00:00:00Z"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/invitations":
			if got := r.URL.Query().Get("email"); got != "user@example.com" {
				t.Fatalf("expected email query, got %q", got)
			}
			fmt.Fprint(w, `{"data":{"id":"inv_1","organizationId":"org_1","userEmail":"user@example.com","status":"PENDING","creationDate":"2026-01-01T00:00:00Z"}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/invitations/inv_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "invitations", "list", "--status", "PENDING")
	if err != nil {
		t.Fatalf("cloud organizations invitations list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "inv_1\torg_1\tuser@example.com\tPENDING") {
		t.Fatalf("unexpected invitations list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "invitations", "send", "user@example.com")
	if err != nil {
		t.Fatalf("cloud organizations invitations send: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud invitation inv_1 sent.\n" {
		t.Fatalf("unexpected invitation send output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "invitations", "delete", "inv_1")
	if err == nil {
		t.Fatal("expected invitation delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "invitations", "delete", "inv_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations invitations delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud invitation inv_1 deleted.\n" {
		t.Fatalf("unexpected invitation delete output: %q", stdout)
	}
}

func TestCloudOrganizationUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/users":
			fmt.Fprint(w, `{"data":[{"id":"user_1","email":"user@example.com","policyId":42}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/users/user_1":
			fmt.Fprint(w, `{"data":{"id":"user_1","email":"user@example.com","policyId":42}}`)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/users/user_1":
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"policyId":42`) {
				t.Fatalf("expected policy id body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/users/user_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "users", "list")
	if err != nil {
		t.Fatalf("cloud organizations users list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "user_1\tuser@example.com\t42") {
		t.Fatalf("unexpected users list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "users", "show", "user_1")
	if err != nil {
		t.Fatalf("cloud organizations users show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ID\tuser_1", "Email\tuser@example.com", "Policy\t42"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected user output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "users", "link", "user_1", "--policy-id", "42")
	if err != nil {
		t.Fatalf("cloud organizations users link: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud organization org_1 user user_1 linked.\n" {
		t.Fatalf("unexpected link output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "users", "unlink", "user_1")
	if err == nil {
		t.Fatal("expected users unlink to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "users", "unlink", "user_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations users unlink: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud organization org_1 user user_1 unlinked.\n" {
		t.Fatalf("unexpected unlink output: %q", stdout)
	}
}

func TestCloudOrganizationPolicies(t *testing.T) {
	policyBody := `{"data":{"id":42,"name":"Admin","description":"Admin policy","organizationId":"org_1","protected":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","scopes":[{"id":7,"label":"ledger:read","protected":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/policies":
			fmt.Fprint(w, `{"data":[{"id":42,"name":"Admin","description":"Admin policy","organizationId":"org_1","protected":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/policies/42":
			fmt.Fprint(w, policyBody)
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/policies":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Admin"`, `"description":"Admin policy"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create policy body to contain %s, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, policyBody)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/policies/42":
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"name":"Admin updated"`) {
				t.Fatalf("expected update policy body, got %s", body)
			}
			fmt.Fprint(w, policyBody)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/policies/42":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/policies/42/scopes/7":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/policies/42/scopes/7":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "list")
	if err != nil {
		t.Fatalf("cloud organizations policies list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "42\tAdmin\tfalse") {
		t.Fatalf("unexpected policies list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "show", "42")
	if err != nil {
		t.Fatalf("cloud organizations policies show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ID\t42", "Name\tAdmin", "Scope\t7\tledger:read"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected policy output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "create", "Admin", "--description", "Admin policy")
	if err != nil {
		t.Fatalf("cloud organizations policies create: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud policy 42 created.\n" {
		t.Fatalf("unexpected policy create output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "update", "42", "--name", "Admin updated")
	if err != nil {
		t.Fatalf("cloud organizations policies update: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud policy 42 updated.\n" {
		t.Fatalf("unexpected policy update output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "add-scope", "42", "7")
	if err != nil {
		t.Fatalf("cloud organizations policies add-scope: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud policy 42 scope 7 added.\n" {
		t.Fatalf("unexpected add-scope output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "remove-scope", "42", "7")
	if err == nil {
		t.Fatal("expected remove-scope to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "remove-scope", "42", "7", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations policies remove-scope: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud policy 42 scope 7 removed.\n" {
		t.Fatalf("unexpected remove-scope output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "delete", "42")
	if err == nil {
		t.Fatal("expected policy delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "organizations", "policies", "delete", "42", "--confirm")
	if err != nil {
		t.Fatalf("cloud organizations policies delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud policy 42 deleted.\n" {
		t.Fatalf("unexpected policy delete output: %q", stdout)
	}
}

func TestCloudRegions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/regions":
			fmt.Fprint(w, `{"data":[{"id":"reg_1","name":"EU","baseUrl":"https://region.example","createdAt":"2026-01-01T00:00:00Z","active":true,"public":false,"agentID":"agent_1","outdated":false,"version":"v1"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/regions/reg_1":
			fmt.Fprint(w, `{"data":{"id":"reg_1","name":"EU","baseUrl":"https://region.example","createdAt":"2026-01-01T00:00:00Z","active":true,"public":false,"agentID":"agent_1","outdated":false,"version":"v1"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/regions":
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"name":"EU Private"`) {
				t.Fatalf("expected region create body, got %s", body)
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"reg_2","name":"EU Private","baseUrl":"https://private.example","createdAt":"2026-01-01T00:00:00Z","active":true,"agentID":"agent_2","outdated":false,"organizationID":"org_1","version":"v1"}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/regions/reg_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "regions", "list")
	if err != nil {
		t.Fatalf("cloud regions list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "reg_1\tEU\ttrue\tfalse") {
		t.Fatalf("unexpected regions list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "regions", "show", "reg_1")
	if err != nil {
		t.Fatalf("cloud regions show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ID\treg_1", "Name\tEU", "Active\ttrue", "Public\tfalse", "BaseURL\thttps://region.example", "Version\tv1"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected region output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "regions", "create", "EU Private")
	if err != nil {
		t.Fatalf("cloud regions create: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud region reg_2 created.\n" {
		t.Fatalf("unexpected region create output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "regions", "delete", "reg_1")
	if err == nil {
		t.Fatal("expected region delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "regions", "delete", "reg_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud regions delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud region reg_1 deleted.\n" {
		t.Fatalf("unexpected region delete output: %q", stdout)
	}
}

func TestCloudAppsLifecycle(t *testing.T) {
	manifestFile := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(manifestFile, []byte("stack:\n  name: prod\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/apps":
			if got := r.URL.Query().Get("organizationId"); got != "org_1" {
				t.Fatalf("expected organization org_1, got %q", got)
			}
			if got := r.URL.Query().Get("pageNumber"); got != "1" {
				t.Fatalf("expected pageNumber 1, got %q", got)
			}
			fmt.Fprint(w, `{"data":{"currentPage":1,"totalCount":1,"items":[{"id":"app_1","name":"prod","currentRun":{"id":"run_1","createdAt":"2026-01-01T00:00:00Z","status":"applied","message":"ok"},"currentConfigurationVersion":{"id":"ver_1","autoQueueRuns":true,"status":"uploaded"}}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/apps/app_1":
			fmt.Fprint(w, `{"data":{"id":"app_1","name":"prod","currentRun":{"id":"run_1","createdAt":"2026-01-01T00:00:00Z","status":"applied","message":"ok"},"currentConfigurationVersion":{"id":"ver_1","autoQueueRuns":true,"status":"uploaded"}}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/apps/app_1/current-state-version":
			fmt.Fprint(w, `{"data":{"stack":{"id":"stack_1","region_id":"reg_1"}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/apps":
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"organizationId":"org_1"`) {
				t.Fatalf("expected create app body to contain organization, got %s", body)
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"app_1","name":"prod"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/apps/app_1/deploy":
			body := readRequestBody(t, r)
			if !strings.Contains(body, "stack:") {
				t.Fatalf("expected manifest body, got %s", body)
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"id":"run_1","createdAt":"2026-01-01T00:00:00Z","status":"planned","message":"queued","configurationVersion":{"id":"ver_1","autoQueueRuns":true,"status":"uploaded"}}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/apps/app_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "list")
	if err != nil {
		t.Fatalf("cloud apps list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "app_1\tprod\tapplied\tver_1") {
		t.Fatalf("unexpected cloud apps list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "show", "app_1")
	if err != nil {
		t.Fatalf("cloud apps show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ID\tapp_1", "Name\tprod", "RunStatus\tapplied", "State\tid\tstack_1"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected cloud app output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "create")
	if err != nil {
		t.Fatalf("cloud apps create: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud app app_1 created.\n" {
		t.Fatalf("unexpected cloud apps create output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "deploy", "app_1", "--file", manifestFile)
	if err != nil {
		t.Fatalf("cloud apps deploy: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud app deployment accepted with run run_1.\n" {
		t.Fatalf("unexpected cloud apps deploy output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "delete", "app_1")
	if err == nil {
		t.Fatal("expected cloud apps delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "delete", "app_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud apps delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud app app_1 deleted.\n" {
		t.Fatalf("unexpected cloud apps delete output: %q", stdout)
	}
}

func TestCloudAppsRunsVersionsAndVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/apps/app_1/runs":
			if got := r.URL.Query().Get("pageSize"); got != "25" {
				t.Fatalf("expected pageSize 25, got %q", got)
			}
			fmt.Fprint(w, `{"data":{"items":[{"id":"run_1","createdAt":"2026-01-01T00:00:00Z","status":"applied","message":"ok","configurationVersion":{"id":"ver_1","autoQueueRuns":true,"status":"uploaded"}}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/runs/run_1":
			fmt.Fprint(w, `{"data":{"id":"run_1","createdAt":"2026-01-01T00:00:00Z","status":"applied","message":"ok","configurationVersion":{"id":"ver_1","autoQueueRuns":true,"status":"uploaded"}}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/runs/run_1/logs":
			fmt.Fprint(w, `{"data":[{"timestamp":"2026-01-01T00:00:00Z","module":"terraform","message":"done","diagnostic":{"severity":"info","summary":"ok","detail":"done"}}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/apps/app_1/versions":
			fmt.Fprint(w, `{"data":{"items":[{"id":"ver_1","autoQueueRuns":true,"status":"uploaded"}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/versions/ver_1" && strings.Contains(r.Header.Get("Accept"), "application/json"):
			fmt.Fprint(w, `{"data":{"id":"ver_1","autoQueueRuns":true,"status":"uploaded"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/versions/ver_1" && r.Header.Get("Accept") == "application/yaml":
			w.Header().Set("Content-Type", "application/yaml")
			fmt.Fprint(w, "stack:\n  name: prod\n")
		case r.Method == http.MethodGet && r.URL.Path == "/versions/ver_1" && r.Header.Get("Accept") == "application/gzip":
			w.Header().Set("Content-Type", "application/gzip")
			fmt.Fprint(w, "archive")
		case r.Method == http.MethodGet && r.URL.Path == "/apps/app_1/variables":
			fmt.Fprint(w, `{"data":{"items":[{"id":"var_1","key":"TOKEN","value":"secret","sensitive":true},{"id":"var_2","key":"ENV","value":"prod","description":"Environment","sensitive":false}]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/apps/app_1/variables":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"key":"TOKEN"`, `"value":"secret"`, `"sensitive":true`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected variable body to contain %s, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"var_1","key":"TOKEN","value":"secret","sensitive":true}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/apps/app_1/variables/var_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s accept=%s", r.Method, r.URL.String(), r.Header.Get("Accept"))
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "runs", "list", "app_1", "--page-size", "25")
	if err != nil {
		t.Fatalf("cloud apps runs list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "run_1\tapplied\tver_1\tok") {
		t.Fatalf("unexpected runs list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "runs", "show", "run_1")
	if err != nil {
		t.Fatalf("cloud apps runs show: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Status\tapplied") || !strings.Contains(stdout, "ConfigurationVersionID\tver_1") {
		t.Fatalf("unexpected run show output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "runs", "logs", "run_1")
	if err != nil {
		t.Fatalf("cloud apps runs logs: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "2026-01-01T00:00:00Z\tterraform\tdone\tinfo") {
		t.Fatalf("unexpected run logs output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "versions", "list", "app_1")
	if err != nil {
		t.Fatalf("cloud apps versions list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "ver_1\tuploaded\ttrue") {
		t.Fatalf("unexpected versions list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "versions", "show", "ver_1")
	if err != nil {
		t.Fatalf("cloud apps versions show: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "ID\tver_1") || !strings.Contains(stdout, "Status\tuploaded") {
		t.Fatalf("unexpected version show output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "versions", "manifest", "ver_1")
	if err != nil {
		t.Fatalf("cloud apps versions manifest: %v stderr=%s", err, stderr)
	}
	if stdout != "stack:\n  name: prod\n" {
		t.Fatalf("unexpected manifest output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "versions", "archive", "show", "ver_1")
	if err != nil {
		t.Fatalf("cloud apps versions archive show: %v stderr=%s", err, stderr)
	}
	if stdout != "archive" {
		t.Fatalf("unexpected archive output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "versions", "show-archive", "ver_1")
	if err != nil {
		t.Fatalf("cloud apps versions show-archive: %v stderr=%s", err, stderr)
	}
	if stdout != "archive" {
		t.Fatalf("unexpected deprecated archive output: %q", stdout)
	}
	if !strings.Contains(stderr, "Command cloud apps versions show-archive has been deprecated, use cloud apps versions archive show") {
		t.Fatalf("expected show-archive deprecation warning, got:\n%s", stderr)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "variables", "list", "app_1")
	if err != nil {
		t.Fatalf("cloud apps variables list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "var_1\tTOKEN\t****") || !strings.Contains(stdout, "var_2\tENV\tprod\tEnvironment") {
		t.Fatalf("unexpected variables list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "variables", "create", "app_1", "--key", "TOKEN", "--value", "secret")
	if err != nil {
		t.Fatalf("cloud apps variables create: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "ID\tvar_1") || !strings.Contains(stdout, "Value\t****") {
		t.Fatalf("unexpected variable create output:\n%s", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "variables", "delete", "app_1", "var_1")
	if err == nil {
		t.Fatal("expected cloud apps variables delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "apps", "--deploy-url", server.URL, "variables", "delete", "app_1", "var_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud apps variables delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud app variable var_1 deleted.\n" {
		t.Fatalf("unexpected variable delete output: %q", stdout)
	}
}

func TestCloudCommandsRejectStackContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("cloud command must reject stack contexts before network calls")
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create stack context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "me", "show")
	if err == nil {
		t.Fatal("expected cloud command to reject stack context")
	}
	if !strings.Contains(err.Error(), "cloud commands require a cloud or cloud-stack context") {
		t.Fatalf("unexpected error: %v stderr=%s", err, stderr)
	}
}

func TestCloudStacksListShowAndDeprecatedAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/_info":
			fmt.Fprint(w, `{"version":"v1.0.0","consoleURL":"https://portal.example"}`)
		case "/organizations/org_1/stacks":
			fmt.Fprint(w, `{"data":[{"id":"stack_1","name":"Production","organizationId":"org_1","uri":"https://stack.example/api","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"auditEnabled":false,"synchronised":true,"modules":[]}]}`)
		case "/organizations/org_1/stacks/stack_1":
			fmt.Fprint(w, `{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":"https://stack.example/api","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "list")
	if err != nil {
		t.Fatalf("cloud stacks list: %v stderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, expected := range []string{"ID", "Dashboard", "Audit Enabled", "stack_1", "Production", "https://portal.example", "eu-west-1", "ACTIVE", "No"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected cloud stacks list output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "show", "stack_1")
	if err != nil {
		t.Fatalf("cloud stacks show: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"ID\tstack_1", "Name\tProduction", "Status\tREADY", "URI\thttps://stack.example/api", "Version\tv3.2.4"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected stack output to contain %q, got:\n%s", expected, stdout)
		}
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "stacks", "list")
	if err != nil {
		t.Fatalf("stacks alias list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command stacks has been deprecated, use cloud stacks") {
		t.Fatalf("expected stacks deprecation warning, got:\n%s", stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud_stacks", "list")
	if err != nil {
		t.Fatalf("cloud_stacks alias list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command cloud_stacks has been deprecated, use cloud stacks") {
		t.Fatalf("expected cloud_stacks deprecation warning, got:\n%s", stderr)
	}
}

func TestCloudStacksListMissingOrganizationIsActionable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("cloud stacks list without organization must not call membership in non-interactive tests")
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "prod",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "list")
	if err == nil {
		t.Fatal("expected cloud stacks list to require an organization")
	}
	if !strings.Contains(err.Error(), "organization id is required; pass --organization or select a cloud-stack profile") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloudStacksMutations(t *testing.T) {
	stackBody := `{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":"https://stack.example/api","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/stacks":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Production"`, `"regionID":"eu-west-1"`, `"version":"v3.2.4"`, `"metadata":{"env":"prod"}`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create body to contain %s, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, stackBody)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/stacks/stack_1":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Renamed"`, `"metadata":{"env":"prod"}`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected update body to contain %s, got %s", expected, body)
				}
			}
			fmt.Fprint(w, stackBody)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/stacks/stack_1":
			if got := r.URL.Query().Get("force"); got != "true" {
				t.Fatalf("expected force=true, got %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/stacks/stack_1/enable":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/stacks/stack_1/disable":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/stacks/stack_1/restore":
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, stackBody)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/stacks/stack_1/upgrade":
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"version":"v3.2.5"`) {
				t.Fatalf("expected upgrade body to contain version, got %s", body)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"cloud", "stacks", "create", "Production",
		"--region", "eu-west-1",
		"--version", "v3.2.4",
		"--metadata", "env=prod",
	)
	if err != nil {
		t.Fatalf("cloud stacks create: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 created.\n" {
		t.Fatalf("unexpected create output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"cloud", "stacks", "update", "stack_1",
		"--name", "Renamed",
		"--metadata", "env=prod",
	)
	if err != nil {
		t.Fatalf("cloud stacks update: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 updated.\n" {
		t.Fatalf("unexpected update output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "delete", "stack_1")
	if err == nil {
		t.Fatal("expected delete to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "delete", "stack_1", "--force", "--confirm")
	if err != nil {
		t.Fatalf("cloud stacks delete: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 deleted.\n" {
		t.Fatalf("unexpected delete output: %q", stdout)
	}

	for _, args := range [][]string{
		{"enable", "stack_1"},
		{"disable", "stack_1", "--confirm"},
		{"restore", "stack_1", "--confirm"},
		{"upgrade", "stack_1", "--version", "v3.2.5", "--confirm"},
	} {
		stdout, stderr, err = executeCommand(t, append([]string{"--config-dir", configDir, "cloud", "stacks"}, args...)...)
		if err != nil {
			t.Fatalf("cloud stacks %v: %v stderr=%s", args, err, stderr)
		}
		if !strings.Contains(stdout, "Cloud stack stack_1") {
			t.Fatalf("unexpected action output for %v: %q", args, stdout)
		}
	}
}

func TestCloudStacksCreatePromptsForNameRegionAndVersion(t *testing.T) {
	stackBody := `{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":"https://stack.example/api","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/regions":
			fmt.Fprint(w, `{"data":[{"id":"eu-west-1","name":"Europe","active":true,"public":true},{"id":"private-1","name":"Private region","active":true,"public":false}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/regions/eu-west-1/versions":
			fmt.Fprint(w, `{"data":[{"name":"v2.2","regionID":"eu-west-1"},{"name":"v3.2-rc","regionID":"eu-west-1"},{"name":"v3.1","regionID":"eu-west-1"},{"name":"v3.2","regionID":"eu-west-1"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/stacks":
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Production"`, `"regionID":"eu-west-1"`, `"version":"v3.2"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create body to contain %s, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, stackBody)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "prod",
		"--cloud-url", server.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommandWithInput(t,
		"Production\n1\n1\n",
		"--config-dir", configDir,
		"--organization", "org_1",
		"cloud", "stacks", "create",
	)
	if err != nil {
		t.Fatalf("cloud stacks create wizard: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	for _, expected := range []string{
		"Enter a name:",
		"Name\tProduction",
		"Please select a region",
		"1. eu-west-1 | Public | Europe",
		"Region\teu-west-1",
		"Please select a version",
		"1. v3.2",
		"2. v3.2-rc",
		"3. v3.1",
		"4. v2.2",
		"Version\tv3.2",
		"Cloud stack stack_1 created.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected wizard output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestCloudStacksCreateWaitsUntilReady(t *testing.T) {
	stackVersionsCalls := 0
	stackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/versions" {
			t.Fatalf("unexpected stack request %s %s", r.Method, r.URL.String())
		}
		stackVersionsCalls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer stackServer.Close()

	progressingBody := fmt.Sprintf(`{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":%q,"regionID":"eu-west-1","version":"v3.2","status":"PROGRESSING","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":false,"stargateEnabled":true,"synchronised":false,"modules":[]}}`, stackServer.URL)
	readyBody := fmt.Sprintf(`{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":%q,"regionID":"eu-west-1","version":"v3.2","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}}`, stackServer.URL)
	getStackCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/stacks":
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, progressingBody)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/stacks/stack_1":
			getStackCalls++
			fmt.Fprint(w, readyBody)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"cloud", "stacks", "create", "Production",
		"--region", "eu-west-1",
		"--version", "v3.2",
	)
	if err != nil {
		t.Fatalf("cloud stacks create waits: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 created.\n" {
		t.Fatalf("unexpected create output: %q", stdout)
	}
	if getStackCalls != 1 {
		t.Fatalf("expected one readiness poll, got %d", getStackCalls)
	}
	if stackVersionsCalls != 1 {
		t.Fatalf("expected one stack /versions fallback attempt, got %d", stackVersionsCalls)
	}
}

func TestCloudStacksCreateAvailabilityUsesStackVersionsFirst(t *testing.T) {
	stackVersionsCalls := 0
	stackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/versions" {
			t.Fatalf("unexpected stack request %s %s", r.Method, r.URL.String())
		}
		stackVersionsCalls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"v2.0.0","health":true}]}`)
	}))
	defer stackServer.Close()

	progressingBody := fmt.Sprintf(`{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":%q,"regionID":"eu-west-1","version":"v3.2","status":"PROGRESSING","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":false,"stargateEnabled":true,"synchronised":false,"modules":[]}}`, stackServer.URL)
	getStackCalls := 0
	membershipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/stacks":
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, progressingBody)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/stacks/stack_1":
			getStackCalls++
			t.Fatalf("membership readiness fallback should not be used after stack /versions succeeds")
		default:
			t.Fatalf("unexpected membership request %s %s", r.Method, r.URL.String())
		}
	}))
	defer membershipServer.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", membershipServer.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"cloud", "stacks", "create", "Production",
		"--region", "eu-west-1",
		"--version", "v3.2",
	)
	if err != nil {
		t.Fatalf("cloud stacks create direct availability: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 created.\n" {
		t.Fatalf("unexpected create output: %q", stdout)
	}
	if stackVersionsCalls != 1 {
		t.Fatalf("expected one stack /versions check, got %d", stackVersionsCalls)
	}
	if getStackCalls != 0 {
		t.Fatalf("expected no membership fallback calls, got %d", getStackCalls)
	}
}

func TestCloudStacksHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/organizations/org_1/logs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if got := r.URL.Query().Get("stackId"); got != "stack_1" {
			t.Fatalf("expected stackId stack_1, got %q", got)
		}
		if got := r.URL.Query().Get("pageSize"); got != "10" {
			t.Fatalf("expected pageSize 10, got %q", got)
		}
		if got := r.URL.Query().Get("action"); got != "stacks.updated" {
			t.Fatalf("expected action stacks.updated, got %q", got)
		}
		fmt.Fprint(w, `{"data":{"pageSize":10,"hasMore":false,"data":[{"seq":"1","organizationId":"org_1","userId":"user_1","action":"stacks.updated","date":"2026-01-01T00:00:00Z","data":{}}]}}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "history", "stack_1", "--action", "stacks.updated")
	if err != nil {
		t.Fatalf("cloud stacks history: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "1\torg_1\tuser_1\tstacks.updated\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected stack history output:\n%s", stdout)
	}
}

func TestCloudStacksUsersAndModules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/stacks/stack_1/users":
			fmt.Fprint(w, `{"data":[{"stackId":"stack_1","userId":"user_1","email":"user@example.com","policyId":42}]}`)
		case r.Method == http.MethodPut && r.URL.Path == "/organizations/org_1/stacks/stack_1/users/user_1":
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"policyId":42`) {
				t.Fatalf("expected policy id body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/stacks/stack_1/users/user_1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/organizations/org_1/stacks/stack_1/modules":
			fmt.Fprint(w, `{"data":[{"name":"ledger","state":"ENABLED","status":"READY","lastStatusUpdate":"2026-01-01T00:00:00Z","lastStateUpdate":"2026-01-01T00:00:00Z"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/org_1/stacks/stack_1/modules":
			if got := r.URL.Query().Get("name"); got != "ledger" {
				t.Fatalf("expected module name ledger, got %q", got)
			}
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodDelete && r.URL.Path == "/organizations/org_1/stacks/stack_1/modules":
			if got := r.URL.Query().Get("name"); got != "ledger" {
				t.Fatalf("expected module name ledger, got %q", got)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud-stack", "prod",
		"--cloud-url", server.URL,
		"--organization", "org_1",
		"--stack", "stack_1",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud-stack context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "users", "list", "stack_1")
	if err != nil {
		t.Fatalf("cloud stacks users list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "user_1\tuser@example.com\tstack_1\t42") {
		t.Fatalf("unexpected users list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "users", "link", "stack_1", "user_1", "--policy-id", "42")
	if err != nil {
		t.Fatalf("cloud stacks users link: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 user user_1 linked.\n" {
		t.Fatalf("unexpected user link output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "users", "unlink", "stack_1", "user_1")
	if err == nil {
		t.Fatal("expected users unlink to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "users", "unlink", "stack_1", "user_1", "--confirm")
	if err != nil {
		t.Fatalf("cloud stacks users unlink: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 user user_1 unlinked.\n" {
		t.Fatalf("unexpected user unlink output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "modules", "list", "stack_1")
	if err != nil {
		t.Fatalf("cloud stacks modules list: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "ledger\tENABLED\tREADY") {
		t.Fatalf("unexpected modules list output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "modules", "enable", "stack_1", "ledger")
	if err != nil {
		t.Fatalf("cloud stacks modules enable: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 module ledger enabled.\n" {
		t.Fatalf("unexpected module enable output: %q", stdout)
	}

	_, _, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "modules", "disable", "stack_1", "ledger")
	if err == nil {
		t.Fatal("expected modules disable to require --confirm")
	}
	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "modules", "disable", "stack_1", "ledger", "--confirm")
	if err != nil {
		t.Fatalf("cloud stacks modules disable: %v stderr=%s", err, stderr)
	}
	if stdout != "Cloud stack stack_1 module ledger disabled.\n" {
		t.Fatalf("unexpected module disable output: %q", stdout)
	}
}

func TestCloudStacksRejectStackContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("cloud stacks command must reject stack contexts before network calls")
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create stack context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "cloud", "stacks", "list", "--organization", "org_1")
	if err == nil {
		t.Fatal("expected cloud stacks command to reject stack context")
	}
	if !strings.Contains(err.Error(), "cloud commands require a cloud or cloud-stack context") {
		t.Fatalf("unexpected error: %v stderr=%s", err, stderr)
	}
}

func TestContextCommandsJSON(t *testing.T) {
	configDir := t.TempDir()

	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost/api",
		"--auth-method", "client_credentials",
		"--issuer-url", "http://localhost/api/auth",
		"--client-id", "testing",
		"--secret-ref", "keyring://local/testing",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "-o", "json", "context", "list")
	if err != nil {
		t.Fatalf("list contexts json: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{`"currentContext": "local"`, `"contexts": [`, `"local"`} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected JSON output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestTargetInspect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL+"/api",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Context: local",
		"Target: " + server.URL + "/api (stack)",
		"- ledger 2.3.4 healthy api=[v1 v2] policy=latest-compatible",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected inspect output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "--debug", "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target with debug: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Context: local") {
		t.Fatalf("expected inspect stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "debug: context=local target=stack url="+server.URL+"/api") {
		t.Fatalf("expected debug stderr, got:\n%s", stderr)
	}
}

func TestTargetInspectJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL+"/api",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "-o", "json", "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target json: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		`"context": "local"`,
		`"targetKind": "stack"`,
		`"name": "ledger"`,
		`"apiVersions": [`,
		`"v2"`,
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected JSON inspect output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestDeferredCloudAndOIDCLoginCommandsReturnActionableErrors(t *testing.T) {
	_, stderr, err := executeCommand(t, "session", "login", "cloud", "--cloud-url", "https://cloud.example")
	if err == nil {
		t.Fatal("expected session login cloud to be deferred")
	}
	if !strings.Contains(err.Error(), "session login cloud is deferred") {
		t.Fatalf("unexpected cloud login error: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "login")
	if err == nil {
		t.Fatal("expected interactive root login without input to fail")
	}
	if !strings.Contains(err.Error(), "unsupported login choice") {
		t.Fatalf("unexpected root login error: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "auth", "login")
	if err == nil {
		t.Fatal("expected auth login alias to be removed")
	}
	if !strings.Contains(err.Error(), `unknown command "login"`) {
		t.Fatalf("unexpected auth login error: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "session", "login", "oidc", "--issuer-url", "https://issuer.example", "--client-id", "client")
	if err == nil {
		t.Fatal("expected session login oidc to be deferred")
	}
	if !strings.Contains(err.Error(), "session login oidc is deferred") {
		t.Fatalf("unexpected oidc login error: %v stderr=%s", err, stderr)
	}
}

func TestSessionLoginTokenStoresCredentialAndUpdatesContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer stack-token" {
			t.Fatalf("expected bearer token, got %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	credentialDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommandWithInput(t,
		"stack-token\n",
		"--config-dir", configDir,
		"--credential-dir", credentialDir,
		"session", "login", "token",
		"--token-stdin",
	)
	if err != nil {
		t.Fatalf("session login token: %v stderr=%s", err, stderr)
	}
	if stdout != "Authentication for context local set to token.\n" {
		t.Fatalf("unexpected session login output: %q", stdout)
	}

	configData, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(configData), "stack-token") {
		t.Fatalf("config should not contain clear token:\n%s", string(configData))
	}
	if !strings.Contains(string(configData), "tokenRef: contexts/local/token") {
		t.Fatalf("expected token ref in config:\n%s", string(configData))
	}
	credentialPath := filepath.Join(credentialDir, "contexts", "local", "token")
	credentialData, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatalf("read stored token: %v", err)
	}
	if string(credentialData) != "stack-token" {
		t.Fatalf("unexpected stored token %q", credentialData)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"session", "status",
	)
	if err != nil {
		t.Fatalf("session status: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Context\tlocal\n",
		"Method\ttoken\n",
		"TokenRef\tcontexts/local/token\n",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected session status output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--credential-dir", credentialDir,
		"session", "token",
	)
	if err != nil {
		t.Fatalf("session token: %v stderr=%s", err, stderr)
	}
	if stdout != "stack-token\n" {
		t.Fatalf("unexpected session token output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--credential-dir", credentialDir,
		"target", "inspect",
	)
	if err != nil {
		t.Fatalf("target inspect with stored token: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Context: local") {
		t.Fatalf("unexpected inspect output:\n%s", stdout)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--credential-dir", credentialDir,
		"session", "logout",
	)
	if err == nil {
		t.Fatal("expected session logout to require confirmation")
	}
	if !strings.Contains(err.Error(), "session logout requires --confirm") {
		t.Fatalf("unexpected logout error: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--credential-dir", credentialDir,
		"session", "logout",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("session logout: %v stderr=%s", err, stderr)
	}
	if stdout != "Authentication for context local cleared.\n" {
		t.Fatalf("unexpected session logout output: %q", stdout)
	}
	if _, err := os.Stat(credentialPath); !os.IsNotExist(err) {
		t.Fatalf("expected stored token to be deleted, got %v", err)
	}
	configData, err = os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("read config after logout: %v", err)
	}
	if !strings.Contains(string(configData), "method: none") || strings.Contains(string(configData), "tokenRef") {
		t.Fatalf("expected session logout to clear token auth:\n%s", string(configData))
	}
	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--credential-dir", credentialDir,
		"session", "token",
	)
	if err == nil {
		t.Fatal("expected session token to require an authenticated context")
	}
	if !strings.Contains(err.Error(), "session token requires an authenticated context") {
		t.Fatalf("unexpected session token error: %v stderr=%s", err, stderr)
	}
}

func TestSessionLoginTokenUsesDefaultCredentialDir(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"session", "login", "token",
		"--token", "stack-token",
	)
	if err != nil {
		t.Fatalf("session login token with default credential dir: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"session", "token",
	)
	if err != nil {
		t.Fatalf("session token with default credential dir: %v stderr=%s", err, stderr)
	}
	if stdout != "stack-token\n" {
		t.Fatalf("unexpected session token output: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(configDir, "credentials", "contexts", "local", "token")); err != nil {
		t.Fatalf("expected token in default credential dir: %v", err)
	}
}

func TestSessionLoginClientCredentialsStoresSecret(t *testing.T) {
	configDir := t.TempDir()
	credentialDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommandWithInput(t,
		"super-secret\n",
		"--config-dir", configDir,
		"--credential-dir", credentialDir,
		"session", "login", "client-credentials",
		"--issuer-url", "http://issuer",
		"--client-id", "client",
		"--client-secret-stdin",
	)
	if err != nil {
		t.Fatalf("session login client-credentials: %v stderr=%s", err, stderr)
	}
	if stdout != "Authentication for context local set to client credentials.\n" {
		t.Fatalf("unexpected session login output: %q", stdout)
	}

	configData, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(configData), "super-secret") {
		t.Fatalf("config should not contain clear client secret:\n%s", string(configData))
	}
	if !strings.Contains(string(configData), "method: client_credentials") ||
		!strings.Contains(string(configData), "secretRef: contexts/local/client-secret") {
		t.Fatalf("expected client credentials config:\n%s", string(configData))
	}
	secretData, err := os.ReadFile(filepath.Join(credentialDir, "contexts", "local", "client-secret"))
	if err != nil {
		t.Fatalf("read stored secret: %v", err)
	}
	if string(secretData) != "super-secret" {
		t.Fatalf("unexpected stored secret %q", secretData)
	}
}

func TestSessionLoginClientCredentialsBootstrapsDefaultCloudContext(t *testing.T) {
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"session", "login", "client-credentials",
		"--issuer-url", "https://app.formance.cloud/api",
		"--client-id", "client",
		"--client-secret", "super-secret",
	)
	if err != nil {
		t.Fatalf("session login client-credentials bootstrap: %v stderr=%s", err, stderr)
	}
	if stdout != "Authentication for context formance-cloud set to client credentials.\n" {
		t.Fatalf("unexpected session login output: %q", stdout)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load bootstrapped config: %v", err)
	}
	context := cfg.Contexts[v4config.DefaultCloudContextName]
	if cfg.CurrentContext != v4config.DefaultCloudContextName ||
		context.Kind != v4config.ContextKindCloud ||
		context.CloudURL != v4config.DefaultCloudURL ||
		context.Auth.Method != v4config.AuthMethodClientCredentials ||
		context.Auth.ClientID != "client" ||
		context.Auth.SecretRef != "contexts/formance-cloud/client-secret" {
		t.Fatalf("unexpected bootstrapped context: current=%q context=%#v", cfg.CurrentContext, context)
	}
	if !stringSliceContains(context.Auth.Scopes, "organization:ListStacks") {
		t.Fatalf("expected cloud client credentials to include organization scopes, got %#v", context.Auth.Scopes)
	}
	secretData, err := os.ReadFile(filepath.Join(configDir, "credentials", "contexts", "formance-cloud", "client-secret"))
	if err != nil {
		t.Fatalf("read stored secret: %v", err)
	}
	if string(secretData) != "super-secret" {
		t.Fatalf("unexpected stored secret %q", secretData)
	}
}

func TestSessionLoginClientCredentialsUsesDefaultFctlConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	stdout, stderr, err := executeCommand(t,
		"session", "login", "client-credentials",
		"--issuer-url", "https://app.formance.cloud/api",
		"--client-id", "client",
		"--client-secret", "super-secret",
	)
	if err != nil {
		t.Fatalf("session login client-credentials with default config dir: %v stderr=%s", err, stderr)
	}
	if stdout != "Authentication for context formance-cloud set to client credentials.\n" {
		t.Fatalf("unexpected session login output: %q", stdout)
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("resolve user config dir: %v", err)
	}
	expectedDir := filepath.Join(userConfigDir, "formance", "fctl")
	if _, err := os.Stat(filepath.Join(expectedDir, "config.yaml")); err != nil {
		t.Fatalf("expected config in default fctl dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(expectedDir, "credentials", "contexts", "formance-cloud", "client-secret")); err != nil {
		t.Fatalf("expected credentials in default fctl dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(userConfigDir, "formance", "fctl-v4", "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("did not expect config in fctl-v4 dir, got %v", err)
	}
}

func TestSessionLoginNoneRequiresConfirmForCloudContext(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", "http://localhost",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "session", "login", "none")
	if err == nil {
		t.Fatal("expected session login none to require confirmation on cloud contexts")
	}
	if !strings.Contains(err.Error(), "session login none on cloud contexts requires --confirm") {
		t.Fatalf("unexpected error: %v stderr=%s", err, stderr)
	}
}

func TestProfileFlagSelectsContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL+"/api",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "--profile", "local", "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target with profile: %v stderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr for primary profile flag, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "Context: local") {
		t.Fatalf("unexpected inspect output:\n%s", stdout)
	}
}

func TestContextFlagSelectsProfileWithDeprecationWarning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/versions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL+"/api",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "--context", "local", "target", "inspect")
	if err != nil {
		t.Fatalf("inspect target with context alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Flag --context has been deprecated, use --profile") {
		t.Fatalf("expected context deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "Context: local") {
		t.Fatalf("unexpected inspect output:\n%s", stdout)
	}
}

func TestProfileAndContextFlagsAreMutuallyExclusive(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "--context", "local", "--profile", "local", "target", "inspect")
	if err == nil {
		t.Fatal("expected mutually exclusive context/profile error")
	}
	if !strings.Contains(err.Error(), "--profile and --context are mutually exclusive") {
		t.Fatalf("unexpected error: %v stderr=%s", err, stderr)
	}
}

func TestTargetProxyRejectsCloudContext(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "cloud", "cloud",
		"--cloud-url", "http://localhost",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "target", "proxy")
	if err == nil {
		t.Fatal("expected target proxy to reject cloud context")
	}
	if !strings.Contains(err.Error(), "stack target commands on Cloud profiles require --organization and --stack") {
		t.Fatalf("unexpected proxy error: %v stderr=%s", err, stderr)
	}
}

func TestStackProxyAliasRoutesToTargetProxy(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"profile", "create", "cloud", "cloud",
		"--cloud-url", "http://localhost:8080",
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud profile: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"stack", "proxy",
	)
	if err == nil {
		t.Fatal("expected stack proxy alias to reach target proxy and reject cloud profile")
	}
	if !strings.Contains(stderr, "Command stack proxy has been deprecated, use target proxy") {
		t.Fatalf("expected stack proxy deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(err.Error(), "stack target commands on Cloud profiles require --organization and --stack") {
		t.Fatalf("unexpected stack proxy error: %v stderr=%s", err, stderr)
	}
}

func TestLedgerListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			if got := r.URL.Query().Get("includeDeleted"); got != "true" {
				t.Fatalf("expected includeDeleted true, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"name":"default","bucket":"bucket","addedAt":"2026-01-01T00:00:00Z","metadata":{"env":"dev"}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "list",
		"--page-size", "10",
		"--include-deleted",
	)
	if err != nil {
		t.Fatalf("list ledgers: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Name",
		"Bucket",
		"Created at",
		"default",
		"bucket",
		"2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger list output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerListCloudStackExchangesStackToken(t *testing.T) {
	var stackServer *httptest.Server
	stackServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"token_endpoint":%q}`, stackServer.URL+"/api/auth/oauth/token")
		case "/api/auth/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse stack token form: %v", err)
			}
			if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" ||
				r.Form.Get("assertion") != "stack-assertion" ||
				r.Form.Get("scope") != "openid email" {
				t.Fatalf("unexpected stack token form: %v", r.Form)
			}
			fmt.Fprint(w, `{"access_token":"stack-token","token_type":"Bearer"}`)
		case "/versions":
			if got := r.Header.Get("Authorization"); got != "Bearer stack-token" {
				t.Fatalf("expected stack token on versions, got %q", got)
			}
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2":
			if got := r.Header.Get("Authorization"); got != "Bearer stack-token" {
				t.Fatalf("expected stack token on ledger list, got %q", got)
			}
			fmt.Fprint(w, `{"cursor":{"data":[{"name":"default","bucket":"bucket","addedAt":"2026-01-01T00:00:00Z"}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected stack path %s", r.URL.Path)
		}
	}))
	defer stackServer.Close()

	membershipTokenRequests := 0
	var membershipServer *httptest.Server
	membershipServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"token_endpoint":%q}`, membershipServer.URL+"/token")
		case "/token":
			clientID, clientSecret, ok := r.BasicAuth()
			if !ok || clientID != "client" || clientSecret != "secret" {
				t.Fatalf("unexpected basic auth: %q %q %v", clientID, clientSecret, ok)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse membership token form: %v", err)
			}
			membershipTokenRequests++
			switch membershipTokenRequests {
			case 1:
				if !strings.Contains(r.Form.Get("scope"), "organization:ReadStack") {
					t.Fatalf("expected organization scopes, got %q", r.Form.Get("scope"))
				}
				fmt.Fprint(w, `{"access_token":"org-token","token_type":"Bearer"}`)
			case 2:
				if r.Form.Get("resource") != "stack://org_1/stack_1|stack:Read stack:Write" {
					t.Fatalf("unexpected stack resource: %q", r.Form.Get("resource"))
				}
				if r.Form.Get("scope") != "openid offline_access" {
					t.Fatalf("unexpected stack assertion scope: %q", r.Form.Get("scope"))
				}
				fmt.Fprint(w, `{"access_token":"stack-assertion","token_type":"Bearer"}`)
			default:
				t.Fatalf("unexpected membership token request %d", membershipTokenRequests)
			}
		case "/organizations/org_1/stacks/stack_1":
			if got := r.Header.Get("Authorization"); got != "Bearer org-token" {
				t.Fatalf("expected org token on stack lookup, got %q", got)
			}
			fmt.Fprintf(w, `{"data":{"id":"stack_1","name":"Production","organizationId":"org_1","uri":%q,"regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}}`, stackServer.URL)
		default:
			t.Fatalf("unexpected membership path %s", r.URL.Path)
		}
	}))
	defer membershipServer.Close()

	configDir := t.TempDir()
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"--organization", "org_1",
		"--stack", "stack_1",
		"login",
		"--target", "cloud",
		"--membership-url", membershipServer.URL,
		"--client-id", "client",
		"--client-secret", "secret",
	)
	if err != nil {
		t.Fatalf("login cloud client credentials: %v stderr=%s", err, stderr)
	}
	if stdout != "Logged in with profile default.\n" {
		t.Fatalf("unexpected login output: %q", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "list",
	)
	if err != nil {
		t.Fatalf("ledger list cloud-stack: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v2", "Name", "Bucket", "default", "bucket"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected cloud-stack ledger list output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerCloudOrganizationAutoSelectsSingleStack(t *testing.T) {
	stackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/versions":
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"1.9.0","health":true}]}`)
		case "/api/ledger/_info":
			fmt.Fprint(w, `{"data":{"server":"ledger","version":"1.9.0","config":{}}}`)
		default:
			t.Fatalf("unexpected stack path %s", r.URL.Path)
		}
	}))
	defer stackServer.Close()

	membershipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/_info":
			fmt.Fprint(w, `{"version":"v1.0.0","consoleURL":"https://portal.example"}`)
		case "/organizations/org_1/stacks":
			fmt.Fprintf(w, `{"data":[{"id":"stack_1","name":"Production","organizationId":"org_1","uri":%q,"regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}]}`, stackServer.URL)
		default:
			t.Fatalf("unexpected membership path %s", r.URL.Path)
		}
	}))
	defer membershipServer.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"profile", "create", "cloud", "cloud",
		"--cloud-url", membershipServer.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud profile: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"--organization", "org_1",
		"ledger", "info",
	)
	if err != nil {
		t.Fatalf("ledger info with single cloud stack: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "Server", "ledger", "Version", "1.9.0"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger info output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerCloudOrganizationRequiresStackWhenMultipleStacks(t *testing.T) {
	membershipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/_info":
			fmt.Fprint(w, `{"version":"v1.0.0","consoleURL":"https://portal.example"}`)
		case "/organizations/org_1/stacks":
			fmt.Fprint(w, `{"data":[{"id":"stack_1","name":"Production","organizationId":"org_1","uri":"https://stack1.example","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]},{"id":"stack_2","name":"Staging","organizationId":"org_1","uri":"https://stack2.example","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}]}`)
		default:
			t.Fatalf("unexpected membership path %s", r.URL.Path)
		}
	}))
	defer membershipServer.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"profile", "create", "cloud", "cloud",
		"--cloud-url", membershipServer.URL,
		"--auth-method", "none",
	)
	if err != nil {
		t.Fatalf("create cloud profile: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"--organization", "org_1",
		"ledger", "info",
	)
	if err == nil {
		t.Fatal("expected multiple stacks to require --stack")
	}
	for _, expected := range []string{
		`--stack is required when organization "org_1" has multiple stacks`,
		"available stacks: stack_1 (Production), stack_2 (Staging)",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to contain %q, got err=%v stderr=%s", expected, err, stderr)
		}
	}
}

func TestLedgerCloudStackMissingStackSuggestsAvailableStacks(t *testing.T) {
	membershipTokenRequests := 0
	var membershipServer *httptest.Server
	membershipServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"token_endpoint":%q}`, membershipServer.URL+"/token")
		case "/token":
			membershipTokenRequests++
			fmt.Fprint(w, `{"access_token":"org-token","token_type":"Bearer"}`)
		case "/organizations/org_1/stacks/missing":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"errorCode":"INTERNAL","errorMessage":"not found"}`)
		case "/_info":
			fmt.Fprint(w, `{"version":"v1.0.0","consoleURL":"https://portal.example"}`)
		case "/organizations/org_1/stacks":
			fmt.Fprint(w, `{"data":[{"id":"stack_1","name":"Production","organizationId":"org_1","uri":"https://stack.example/api","regionID":"eu-west-1","version":"v3.2.4","status":"READY","state":"ACTIVE","expectedStatus":"READY","lastStateUpdate":"2026-01-01T00:00:00Z","lastExpectedStatusUpdate":"2026-01-01T00:00:00Z","lastStatusUpdate":"2026-01-01T00:00:00Z","reachable":true,"stargateEnabled":true,"synchronised":true,"modules":[]}]}`)
		default:
			t.Fatalf("unexpected membership path %s", r.URL.Path)
		}
	}))
	defer membershipServer.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"--organization", "org_1",
		"--stack", "missing",
		"login",
		"--target", "cloud",
		"--membership-url", membershipServer.URL,
		"--client-id", "client",
		"--client-secret", "secret",
	)
	if err != nil {
		t.Fatalf("login cloud client credentials: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "ledger", "info")
	if err == nil {
		t.Fatal("expected missing stack to fail")
	}
	for _, expected := range []string{
		`stack "missing" was not found in organization "org_1"`,
		"available stacks: stack_1 (Production)",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to contain %q, got err=%v stderr=%s", expected, err, stderr)
		}
	}
	if membershipTokenRequests == 0 {
		t.Fatal("expected membership token request")
	}
}

func TestLedgerInfoUsesCanonicalCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"1.9.0","health":true}]}`)
		case "/api/ledger/_info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"server":"ledger","version":"1.9.0","config":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "info")
	if err != nil {
		t.Fatalf("read ledger info: %v stderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected canonical command to keep stderr empty, got:\n%s", stderr)
	}
	for _, expected := range []string{
		"API version: v1",
		"Server",
		"ledger",
		"Version",
		"1.9.0",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected info output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerServerInfosDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"1.9.0","health":true}]}`)
		case "/api/ledger/_info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"server":"ledger","version":"1.9.0","config":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "ledger", "server-infos")
	if err != nil {
		t.Fatalf("read ledger info through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use ledger info") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestLedgerCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "ledger", "create", "default")
	if err == nil {
		t.Fatal("expected ledger create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "ledger create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestLedgerCreateSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"bucket":"bucket-a"`,
				`"features":{"hash":"true"}`,
				`"metadata":{"tier":"gold"}`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected body to contain %q, got %s", expected, body)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "create", "default",
		"--bucket", "bucket-a",
		"--feature", "hash=true",
		"--metadata", "tier=gold",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create ledger: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Ledger default created.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerSetMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/metadata":
			if r.Method != http.MethodPut {
				t.Fatalf("expected PUT, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"tier":"gold"`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			if !strings.Contains(body, `"team":"finance"`) {
				t.Fatalf("expected metadata file body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	metadataPath := filepath.Join(t.TempDir(), "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(`{"team":"finance","tier":"silver"}`), 0o600); err != nil {
		t.Fatalf("write metadata fixture: %v", err)
	}
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "set-metadata", "default", "tier=gold",
		"--metadata-file", metadataPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set ledger metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata added.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerDeleteMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/metadata/tier":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "delete-metadata", "default", "tier",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete ledger metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerExportSelectsV2ToFile(t *testing.T) {
	export := "log-entry-1\nlog-entry-2\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/export":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			fmt.Fprint(w, export)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	outputPath := filepath.Join(t.TempDir(), "export.jsonl")
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "export",
		"--file", outputPath,
	)
	if err != nil {
		t.Fatalf("export ledger: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Ledger default exported to " + outputPath + ".",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if string(data) != export {
		t.Fatalf("unexpected export data: %q", string(data))
	}
}

func TestLedgerExportSelectsV2ToStdout(t *testing.T) {
	export := "log-entry-1\nlog-entry-2\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/export":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			fmt.Fprint(w, export)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "export",
		"--file", "-",
	)
	if err != nil {
		t.Fatalf("export ledger: %v stderr=%s", err, stderr)
	}
	if stdout != export {
		t.Fatalf("unexpected stdout export: %q", stdout)
	}
}

func TestLedgerImportSelectsV2FromFile(t *testing.T) {
	input := "{\"id\":1}\n{\"id\":2}\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/import":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if body != input {
				t.Fatalf("unexpected import body: %q", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	inputPath := filepath.Join(t.TempDir(), "import.jsonl")
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatalf("write import file: %v", err)
	}
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "import", "default",
		"--file", inputPath,
	)
	if err != nil {
		t.Fatalf("import ledger: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Ledger default imported.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerImportAcceptsDeprecatedPositionalFile(t *testing.T) {
	input := "{\"id\":1}\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/logs/import":
			if body := readRequestBody(t, r); body != input {
				t.Fatalf("unexpected import body: %q", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	inputPath := filepath.Join(t.TempDir(), "import.jsonl")
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatalf("write import file: %v", err)
	}
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "import", "default", inputPath,
	)
	if err != nil {
		t.Fatalf("import ledger: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Ledger default imported.") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
}

func TestLedgerSchemasListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/schemas":
			if got := r.URL.Query().Get("pageSize"); got != "15" {
				t.Fatalf("expected pageSize 15, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{}}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "schemas", "list")
	if err != nil {
		t.Fatalf("list schemas: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v2", "Version", "Created at", "Chart segments", "v1", "2026-01-01T00:00:00Z"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected schemas output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerSchemasShowSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/schemas/v1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"version":"v1","createdAt":"2026-01-01T00:00:00Z","chart":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "schemas", "show", "v1")
	if err != nil {
		t.Fatalf("show schema: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Version",
		"v1",
		"Created at",
		"2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerSchemasInsertSelectsV2(t *testing.T) {
	schema := `{"chart":{}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/schemas/v1":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "schema-key" {
				t.Fatalf("expected idempotency key, got %q", got)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"chart":{}`) {
				t.Fatalf("expected schema body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	schemaPath := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(schemaPath, []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema file: %v", err)
	}
	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "schemas", "insert", "v1",
		"--file", schemaPath,
		"--idempotency-key", "schema-key",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("insert schema: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Schema v1 inserted in ledger default.") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
}

func TestLedgerAccountsListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts":
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"address":"users:123"`) {
				t.Fatalf("expected query to contain account address, got %q", query)
			}
			if !strings.Contains(query, `"metadata[tier]":"gold"`) {
				t.Fatalf("expected query to contain metadata filter, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"address":"users:123","metadata":{"tier":"gold"}}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "list",
		"--account", "users:123",
		"--metadata", "tier=gold",
	)
	if err != nil {
		t.Fatalf("list accounts: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "users:123") {
		t.Fatalf("unexpected accounts output:\n%s", stdout)
	}
}

func TestLedgerAccountsListDeprecatedAddressAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "list",
		"--address", "users:123",
	)
	if err != nil {
		t.Fatalf("list accounts with deprecated alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Flag --address has been deprecated, use --account") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestLedgerAccountsQuerySelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/queries/active_accounts/run":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.URL.Query().Get("schemaVersion"); got != "v1" {
				t.Fatalf("expected schemaVersion v1, got %q", got)
			}
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			if got := r.URL.Query().Get("sort"); got != "address:asc" {
				t.Fatalf("expected sort address:asc, got %q", got)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"resource":"accounts"`) {
				t.Fatalf("expected account query params, got %s", body)
			}
			if !strings.Contains(body, `"segment":"vip"`) {
				t.Fatalf("expected query variable, got %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"address":"users:123","metadata":{"segment":"vip"}}],"hasMore":true,"pageSize":10,"next":"next"},"resource":"accounts"}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "query", "active_accounts",
		"--schema-version", "v1",
		"--page-size", "10",
		"--sort", "address:asc",
		"--var", "segment=vip",
	)
	if err != nil {
		t.Fatalf("run account query: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Query",
		"active_accounts",
		"Address",
		"users:123",
		"Next: next",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected account query output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerAccountsQueryRequiresSchemaVersion(t *testing.T) {
	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", "http://localhost",
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "query", "active_accounts",
	)
	if err == nil {
		t.Fatal("expected account query without schema version to fail")
	}
	if !strings.Contains(err.Error(), "schema version is required") {
		t.Fatalf("expected schema version error, got err=%v stderr=%s", err, stderr)
	}
}

func TestLedgerAccountsShowSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts/users:123":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"address":"users:123","metadata":{"tier":"gold"},"volumes":{"USD/2":{"input":100,"output":40,"balance":60}}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "accounts", "show", "users:123")
	if err != nil {
		t.Fatalf("show account: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Address",
		"users:123",
		"Asset",
		"Input",
		"Output",
		"Balance",
		"USD/2",
		"100",
		"40",
		"60",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerAccountsSetMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts/users:123/metadata":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"foo":"bar"`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "set-metadata", "users:123", "foo=bar",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set account metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata added.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerAccountsDeleteMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/accounts/users:123/metadata/foo":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "accounts", "delete-metadata", "users:123", "foo",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete account metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerStatsSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/stats":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"accounts":2,"transactions":42}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "stats")
	if err != nil {
		t.Fatalf("read stats: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Transactions",
		"42",
		"Accounts",
		"2",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected stats output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			if got := r.URL.Query().Get("pageSize"); got != "15" {
				t.Fatalf("expected pageSize 15, got %q", got)
			}
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"source":"world"`) {
				t.Fatalf("expected query to contain source world, got %q", query)
			}
			if !strings.Contains(query, `"destination":"users:123"`) {
				t.Fatalf("expected query to contain destination users:123, got %q", query)
			}
			if !strings.Contains(query, `"metadata[tier]":"gold"`) {
				t.Fatalf("expected query to contain metadata tier, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":1,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "list",
		"--source", "world",
		"--destination", "users:123",
		"--metadata", "tier=gold",
	)
	if err != nil {
		t.Fatalf("list transactions: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID",
		"Reference",
		"Timestamp",
		"1",
		"ref",
		"2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsListTimeFiltersSelectV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/default/transactions":
			if got := r.URL.Query().Get("startTime"); got != "2026-01-01T00:00:00Z" {
				t.Fatalf("expected startTime, got %q", got)
			}
			if got := r.URL.Query().Get("endTime"); got != "2026-01-02T00:00:00Z" {
				t.Fatalf("expected endTime, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"txid":1,"metadata":{"foo":"bar"},"postings":[],"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "list",
		"--start", "2026-01-01T00:00:00Z",
		"--end", "2026-01-02T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("list transactions: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") {
		t.Fatalf("expected v1 output, got:\n%s", stdout)
	}
}

func TestLedgerTransactionsCountSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			if r.Method != http.MethodHead {
				t.Fatalf("expected HEAD, got %s", r.Method)
			}
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"source":"world"`) {
				t.Fatalf("expected query to contain source world, got %q", query)
			}
			if !strings.Contains(query, `"destination":"users:123"`) {
				t.Fatalf("expected query to contain destination users:123, got %q", query)
			}
			w.Header().Set("Count", "42")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "count",
		"--source", "world",
		"--destination", "users:123",
	)
	if err != nil {
		t.Fatalf("count transactions: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Count",
		"42",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsExplainRequiresV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, _, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "explain", "42",
	)
	if err == nil {
		t.Fatal("expected ledger transactions explain to require ledger API v3")
	}
	for _, expected := range []string{
		"ledger transactions explain requires ledger API v3+",
		"target ledger component 2.3.4 supports v1,v2",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to contain %q, got %v", expected, err)
		}
	}
}

func TestLedgerTransactionsExplainDocumentsCurrentSDKGapOnV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"3.2.4","health":true}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, _, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "explain", "42",
	)
	if err == nil {
		t.Fatal("expected ledger transactions explain to report the current SDK/spec gap")
	}
	for _, expected := range []string{
		"ledger transactions explain resolved v3",
		"explainTransaction is not exposed by the current stack v3.2.4 spec or formance-sdk-go yet",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to contain %q, got %v", expected, err)
		}
	}
}

func TestLedgerTransactionsSendSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"amount":100`,
				`"asset":"USD/2"`,
				`"source":"world"`,
				`"destination":"users:123"`,
				`"metadata":{"foo":"bar"}`,
				`"reference":"ref"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":42,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "send",
		"--source", "world",
		"--destination", "users:123",
		"--amount", "100",
		"--asset", "USD/2",
		"--metadata", "foo=bar",
		"--reference", "ref",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("send transaction: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID",
		"42",
		"Reference",
		"ref",
		"Timestamp",
		"2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsRunScriptSelectsV2(t *testing.T) {
	scriptFile := filepath.Join(t.TempDir(), "script.num")
	if err := os.WriteFile(scriptFile, []byte("send [COIN 100] (\n  source = @world\n  destination = @user\n)\n"), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"metadata":{"foo":"bar"}`,
				`"reference":"ref"`,
				`"timestamp":"2026-01-01T00:00:00Z"`,
				`"script":`,
				`"plain":"send [COIN 100]`,
				`"vars":{"amount":"100/USD/2","destination":"users:123","portion":"1/2"}`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":43,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, _, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "run-script",
		"--file", scriptFile,
	)
	if err == nil {
		t.Fatal("expected run-script to require --confirm")
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "run-script",
		"--file", scriptFile,
		"--account-var", "destination=users:123",
		"--amount-var", "amount=100/USD/2",
		"--portion-var", "portion=1/2",
		"--metadata", "foo=bar",
		"--reference", "ref",
		"--timestamp", "2026-01-01T00:00:00Z",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("run script: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID",
		"43",
		"Reference",
		"ref",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsNumDeprecatedAlias(t *testing.T) {
	scriptFile := filepath.Join(t.TempDir(), "script.num")
	if err := os.WriteFile(scriptFile, []byte("send [COIN 100] (\n  source = @world\n  destination = @user\n)\n"), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"script":`) {
				t.Fatalf("expected script body, got %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":43,"metadata":{},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "num",
		scriptFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("run script through deprecated num alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command ledger transactions num has been deprecated, use ledger transactions run-script --file <path>|-") {
		t.Fatalf("expected num deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Positional file has been deprecated") {
		t.Fatalf("expected positional deprecation warning, got:\n%s", stderr)
	}
}

func TestLedgerSendDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"source":"world"`,
				`"destination":"users:123"`,
				`"amount":100`,
				`"asset":"USD/2"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":42,"metadata":{},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "send",
		"users:123", "100", "USD/2",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("send through deprecated alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use ledger transactions send") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Positional ledger send arguments have been deprecated") {
		t.Fatalf("expected positional deprecation warning, got:\n%s", stderr)
	}
}

func TestLedgerTransactionsShowSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":42,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"ref"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "ledger", "transactions", "show", "42")
	if err != nil {
		t.Fatalf("show transaction: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID",
		"42",
		"Reference",
		"ref",
		"Timestamp",
		"2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected transaction output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsRevertRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "ledger", "transactions", "revert", "42")
	if err == nil {
		t.Fatal("expected revert to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "ledger transactions revert requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestLedgerTransactionsRevertSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42/revert":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.URL.Query().Get("atEffectiveDate"); got != "true" {
				t.Fatalf("expected atEffectiveDate true, got %q", got)
			}
			if got := r.URL.Query().Get("force"); got != "true" {
				t.Fatalf("expected force true, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":43,"metadata":{"foo":"bar"},"postings":[],"reverted":false,"timestamp":"2026-01-01T00:00:00Z","reference":"revert-ref"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "revert", "42",
		"--at-effective-date",
		"--force",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("revert transaction: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"ID",
		"43",
		"Reference",
		"revert-ref",
		"Timestamp",
		"2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected transaction output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsSetMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42/metadata":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"foo":"bar"`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "set-metadata", "42", "foo=bar",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set transaction metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata added.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerTransactionsDeleteMetadataSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions/42/metadata/foo":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "delete-metadata", "42", "foo",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete transaction metadata: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Metadata deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerVolumesListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/volumes":
			if got := r.URL.Query().Get("startTime"); got != "2026-01-01T00:00:00Z" {
				t.Fatalf("expected startTime, got %q", got)
			}
			if got := r.URL.Query().Get("endTime"); got != "2026-01-02T00:00:00Z" {
				t.Fatalf("expected endTime, got %q", got)
			}
			if got := r.URL.Query().Get("insertionDate"); got != "true" {
				t.Fatalf("expected insertionDate true, got %q", got)
			}
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"account":"users:123"`) {
				t.Fatalf("expected query to contain account, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"account":"users:123","asset":"USD/2","input":100,"output":40,"balance":60}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"ledger", "volumes", "list",
		"--account", "users:123",
		"--start-time", "2026-01-01T00:00:00Z",
		"--end-time", "2026-01-02T00:00:00Z",
		"--use-insertion-date",
	)
	if err != nil {
		t.Fatalf("list volumes: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v2",
		"Account",
		"Asset",
		"Input",
		"Output",
		"Balance",
		"users:123",
		"USD/2",
		"100",
		"40",
		"60",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected volumes output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestLedgerVolumesListDeprecatedAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/volumes":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "volumes", "list",
		"--address", "users:123",
		"--oot", "2026-01-01T00:00:00Z",
		"--pit", "2026-01-02T00:00:00Z",
		"--insertion-date",
	)
	if err != nil {
		t.Fatalf("list volumes with deprecated aliases: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Flag --address has been deprecated, use --account",
		"Flag --oot has been deprecated, use --start-time",
		"Flag --pit has been deprecated, use --end-time",
		"Flag --insertion-date has been deprecated, use --use-insertion-date",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("expected stderr to contain %q, got:\n%s", expected, stderr)
		}
	}
}

func TestLedgerTransactionsListDeprecatedSourceDestinationAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			query := r.URL.Query().Get("query")
			if !strings.Contains(query, `"source":"world"`) {
				t.Fatalf("expected query to contain source world, got %q", query)
			}
			if !strings.Contains(query, `"destination":"users:123"`) {
				t.Fatalf("expected query to contain destination users:123, got %q", query)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
		"--default-ledger", "default",
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"ledger", "transactions", "list",
		"--src", "world",
		"--dst", "users:123",
	)
	if err != nil {
		t.Fatalf("list transactions with deprecated aliases: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Flag --src has been deprecated, use --source",
		"Flag --dst has been deprecated, use --destination",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("expected stderr to contain %q, got:\n%s", expected, stderr)
		}
	}
}

func TestLedgerTransactionsListJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"ledger","version":"2.3.4","health":true}]}`)
		case "/api/ledger/v2/default/transactions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "-o", "json", "ledger", "transactions", "list")
	if err != nil {
		t.Fatalf("list transactions json: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		`"apiVersion": "v2"`,
		`"transactions": []`,
		`"pageSize": 15`,
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected ledger JSON output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsAccountsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{"env":"dev"},"raw":{},"name":"Main"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list payment accounts: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"acc_1\tref\t2026-01-01T00:00:00Z\tMain\tUSD/2\tconn_1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payments accounts output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsVersionsShowsPaymentsComponent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true},{"name":"ledger","version":"2.3.4","health":true}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "payments", "versions")
	if err != nil {
		t.Fatalf("show payments versions: %v stderr=%s", err, stderr)
	}
	expected := "payments 3.1.0 healthy api=[v1 v3] policy=latest-compatible"
	if !strings.Contains(stdout, expected) {
		t.Fatalf("expected payments versions output to contain %q, got:\n%s", expected, stdout)
	}
}

func TestPaymentsAccountsCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "accounts", "create", "--file", "request.json")
	if err == nil {
		t.Fatal("expected payments accounts create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments accounts create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsAccountsCreateSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"accountName":"Main"`,
				`"connectorID":"conn_1"`,
				`"defaultAsset":"USD/2"`,
				`"type":"INTERNAL"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create account body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{"env":"dev"},"raw":{},"name":"Main"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "account.json")
	if err := os.WriteFile(requestFile, []byte(`{"accountName":"Main","connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","defaultAsset":"USD/2","metadata":{"env":"dev"},"reference":"ref","type":"internal"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "create",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create payment account: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Account created with ID: acc_1") {
		t.Fatalf("unexpected create account output:\n%s", stdout)
	}
}

func TestPaymentsAccountsCreateDeprecatedPositionalFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{},"raw":{},"name":"Main"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "account.json")
	if err := os.WriteFile(requestFile, []byte(`{"accountName":"Main","connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","type":"INTERNAL"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "create", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create payment account with positional file: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments accounts create --file <path>|-") {
		t.Fatalf("expected positional file deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsAccountsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts/acc_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{"env":"dev"},"raw":{},"name":"Main"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "show", "acc_1",
	)
	if err != nil {
		t.Fatalf("show payment account: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tacc_1",
		"Reference\tref",
		"Name\tMain",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsAccountsGetDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts/acc_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"acc_1","reference":"ref","createdAt":"2026-01-01T00:00:00Z","connectorID":"conn_1","defaultAsset":"USD/2","provider":"stripe","type":"INTERNAL","metadata":{},"raw":{},"name":"Main"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "get", "acc_1",
	)
	if err != nil {
		t.Fatalf("show payment account through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments accounts show") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsAccountsBalancesSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/accounts/acc_1/balances":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			if got := r.URL.Query().Get("asset"); got != "USD/2" {
				t.Fatalf("expected asset USD/2, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"accountID":"acc_1","asset":"USD/2","balance":100,"createdAt":"2026-01-01T00:00:00Z","lastUpdatedAt":"2026-01-02T00:00:00Z"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "accounts", "balances", "acc_1",
		"--asset", "USD/2",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list account balances: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"acc_1\tUSD/2\t100\t2026-01-01T00:00:00Z\t2026-01-02T00:00:00Z",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected account balances output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsBankAccountsCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "bank-accounts", "create", "--file", "request.json")
	if err == nil {
		t.Fatal("expected payments bank-accounts create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments bank-accounts create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsBankAccountsCreateSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"name":"Main"`,
				`"country":"FR"`,
				`"iban":"FR7630006000011234567890189"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create bank account body to contain %q, got %s", expected, body)
				}
			}
			if strings.Contains(body, "connectorID") {
				t.Fatalf("v3 bank account create body must not contain connectorID, got %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":"ba_1"}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "bank-account.json")
	if err := os.WriteFile(requestFile, []byte(`{"accountNumber":"123","connectorID":"conn_1","country":"FR","iban":"FR7630006000011234567890189","metadata":{"env":"dev"},"name":"Main","swiftBicCode":"AGRIFRPP"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "create",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create payment bank account: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Bank account created with ID: ba_1") {
		t.Fatalf("unexpected create bank account output:\n%s", stdout)
	}
}

func TestPaymentsBankAccountsCreateDeprecatedAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":"ba_1"}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "bank-account.json")
	if err := os.WriteFile(requestFile, []byte(`{"country":"FR","name":"Main"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank_accounts", "create", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create payment bank account with aliases: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments bank-accounts") || !strings.Contains(stderr, "use payments bank-accounts create --file <path>|-") {
		t.Fatalf("expected bank account deprecation warnings, got:\n%s", stderr)
	}
}

func TestPaymentsBankAccountsForwardSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts/ba_1/forward":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"connectorID":"conn_1"`) {
				t.Fatalf("expected connector body, got %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"taskID":"task_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t, "--config-dir", configDir, "context", "create", "stack", "local", "--stack-url", server.URL)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "forward", "ba_1", "conn_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("forward payment bank account: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Bank account forwarding scheduled with task ID: task_1") {
		t.Fatalf("unexpected forward bank account output:\n%s", stdout)
	}
}

func TestPaymentsBankAccountsForwardRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "bank-accounts", "forward", "ba_1", "conn_1")
	if err == nil {
		t.Fatal("expected payments bank-accounts forward to require confirmation")
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("expected empty output, got stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "payments bank-accounts forward requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsBankAccountsSetMetadataSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts/ba_1/metadata":
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"metadata":{"env":"dev"}`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t, "--config-dir", configDir, "context", "create", "stack", "local", "--stack-url", server.URL)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "set-metadata", "ba_1", "env=dev",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set payment bank account metadata: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Metadata set on bank account ba_1.") {
		t.Fatalf("unexpected set bank account metadata output:\n%s", stdout)
	}
}

func TestPaymentsBankAccountsUpdateMetadataDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts/ba_1/metadata":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t, "--config-dir", configDir, "context", "create", "stack", "local", "--stack-url", server.URL)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "update-metadata", "ba_1", "env=dev",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set payment bank account metadata through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments bank-accounts set-metadata") {
		t.Fatalf("expected update-metadata deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsBankAccountsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"ba_1","name":"Main","createdAt":"2026-01-01T00:00:00Z","country":"FR","metadata":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list bank accounts: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "ba_1\tMain\t2026-01-01T00:00:00Z\tFR") {
		t.Fatalf("unexpected bank accounts output:\n%s", stdout)
	}
}

func TestPaymentsBankAccountsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts/ba_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"ba_1","name":"Main","createdAt":"2026-01-01T00:00:00Z","country":"FR","iban":"FR7630006000011234567890189","swiftBicCode":"BNPAFRPP","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank-accounts", "show", "ba_1",
	)
	if err != nil {
		t.Fatalf("show bank account: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tba_1",
		"Country\tFR",
		"IBAN\tFR7630006000011234567890189",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected bank account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsBankAccountsDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/bank-accounts":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "bank_accounts", "list",
	)
	if err != nil {
		t.Fatalf("list bank accounts through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments bank-accounts") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsPaymentsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payments":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"pay_1","reference":"ref","type":"PAYOUT","status":"SUCCEEDED","scheme":"visa","asset":"USD/2","amount":100,"initialAmount":100,"connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","provider":"stripe","metadata":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "payments", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list payments: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "pay_1\tPAYOUT\t100\tUSD/2\tSUCCEEDED\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected payments output:\n%s", stdout)
	}
}

func TestPaymentsPaymentsCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "payments", "create", "--file", "request.json")
	if err == nil {
		t.Fatal("expected payments payments create to require confirmation")
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("expected empty output, got stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "payments payments create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsPaymentsCreateSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payments":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"amount":100`,
				`"initialAmount":100`,
				`"asset":"USD/2"`,
				`"connectorID":"conn_1"`,
				`"type":"PAYOUT"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create payment body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"pay_1","reference":"ref","type":"PAYOUT","status":"SUCCEEDED","scheme":"visa","asset":"USD/2","amount":100,"initialAmount":100,"connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","provider":"stripe","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t, "--config-dir", configDir, "context", "create", "stack", "local", "--stack-url", server.URL)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "payment.json")
	if err := os.WriteFile(requestFile, []byte(`{"amount":100,"asset":"USD/2","connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","metadata":{"env":"dev"},"reference":"ref","scheme":"visa","type":"PAYOUT","status":"SUCCEEDED"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "payments", "create",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create payment: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Payment created with ID: pay_1") {
		t.Fatalf("unexpected create payment output:\n%s", stdout)
	}
}

func TestPaymentsPaymentsCreateDeprecatedPositionalFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payments":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"pay_1","reference":"ref","type":"PAYOUT","status":"SUCCEEDED","scheme":"visa","asset":"USD/2","amount":100,"initialAmount":100,"connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","provider":"stripe","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t, "--config-dir", configDir, "context", "create", "stack", "local", "--stack-url", server.URL)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "payment.json")
	if err := os.WriteFile(requestFile, []byte(`{"amount":100,"asset":"USD/2","connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","reference":"ref","scheme":"visa","type":"PAYOUT","status":"SUCCEEDED"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "payments", "create", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create payment with positional file: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments payments create --file <path>|-") {
		t.Fatalf("expected positional file deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsPaymentsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payments/pay_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"pay_1","reference":"ref","type":"PAYOUT","status":"SUCCEEDED","scheme":"visa","asset":"USD/2","amount":100,"initialAmount":100,"connectorID":"conn_1","createdAt":"2026-01-01T00:00:00Z","provider":"stripe","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "payments", "show", "pay_1",
	)
	if err != nil {
		t.Fatalf("show payment: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tpay_1",
		"Reference\tref",
		"Amount\t100",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPaymentsSetMetadataSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payments/pay_1/metadata":
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"metadata":{"env":"dev"}`) {
				t.Fatalf("expected metadata body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t, "--config-dir", configDir, "context", "create", "stack", "local", "--stack-url", server.URL)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "payments", "set-metadata", "pay_1", "env=dev",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("set payment metadata: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Metadata set on payment pay_1.") {
		t.Fatalf("unexpected set payment metadata output:\n%s", stdout)
	}
}

func TestPaymentsPaymentsSetMetadataRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "payments", "set-metadata", "pay_1", "env=dev")
	if err == nil {
		t.Fatal("expected payments payments set-metadata to require confirmation")
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("expected empty output, got stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "payments payments set-metadata requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsPoolsCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "pools", "create", "--file", "request.json")
	if err == nil {
		t.Fatal("expected payments pools create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments pools create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsPoolsCreateSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Main"`, `"accountIDs":["acc_1"]`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create pool body to contain %q, got:\n%s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":"pool_1"}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}
	requestFile := filepath.Join(t.TempDir(), "pool.json")
	if err := os.WriteFile(requestFile, []byte(`{"name":"Main","accountIDs":["acc_1"]}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "create",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Pool created with ID: pool_1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool create output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"pool_1","name":"Main","poolAccounts":["acc_1"],"createdAt":"2026-01-01T00:00:00Z","type":"STATIC","query":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list payment pools: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "pool_1\tMain\tacc_1") {
		t.Fatalf("unexpected payment pools output:\n%s", stdout)
	}
}

func TestPaymentsPoolsShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"pool_1","name":"Main","poolAccounts":["acc_1","acc_2"],"createdAt":"2026-01-01T00:00:00Z","type":"STATIC","query":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "show", "pool_1",
	)
	if err != nil {
		t.Fatalf("show payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tpool_1",
		"Name\tMain",
		"Accounts\tacc_1,acc_2",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsGetDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"pool_1","name":"Main","poolAccounts":["acc_1"],"createdAt":"2026-01-01T00:00:00Z","type":"STATIC","query":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "get", "pool_1",
	)
	if err != nil {
		t.Fatalf("show payment pool through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments pools show") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsPoolsDeleteRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "pools", "delete", "pool_1")
	if err == nil {
		t.Fatal("expected payments pools delete to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments pools delete requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsPoolsDeleteSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "delete", "pool_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Pool pool_1 deleted.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool delete output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsAddAccountSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/accounts/acc_1":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "add-account", "pool_1", "acc_1",
	)
	if err != nil {
		t.Fatalf("add account to payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Account acc_1 added to pool pool_1.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool add-account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsRemoveAccountRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "pools", "remove-account", "pool_1", "acc_1")
	if err == nil {
		t.Fatal("expected payments pools remove-account to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments pools remove-account requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsPoolsRemoveAccountSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/accounts/acc_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "remove-account", "pool_1", "acc_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("remove account from payment pool: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Account acc_1 removed from pool pool_1.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool remove-account output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsUpdateQuerySelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/query":
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"query":{"accountID":"acc_1"}`) {
				t.Fatalf("expected query body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	queryPath := filepath.Join(t.TempDir(), "query.json")
	if err := os.WriteFile(queryPath, []byte(`{"query":{"accountID":"acc_1"}}`), 0o600); err != nil {
		t.Fatalf("write query fixture: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "update-query", "pool_1",
		"--file", queryPath,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("update pool query: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Query updated for pool pool_1.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment pool update-query output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsPoolsBalancesSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/balances":
			if got := r.URL.Query().Get("at"); got == "" {
				t.Fatal("expected at query parameter")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"asset":"USD/2","amount":100,"relatedAccounts":["acc_1"]}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "balances", "pool_1",
		"--at", "2026-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("list pool balances: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "USD/2\t100\tacc_1") {
		t.Fatalf("unexpected pool balances output:\n%s", stdout)
	}
}

func TestPaymentsPoolsLatestBalancesSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/pools/pool_1/balances/latest":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"asset":"USD/2","amount":100,"relatedAccounts":["acc_1"]}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "pools", "latest-balances", "pool_1",
	)
	if err != nil {
		t.Fatalf("list latest pool balances: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "USD/2\t100\tacc_1") {
		t.Fatalf("unexpected latest pool balances output:\n%s", stdout)
	}
}

func TestPaymentsTasksShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/tasks/task_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"task_1","connectorID":"conn_1","createdObjectID":"pay_1","status":"SUCCEEDED","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:01:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "tasks", "show", "task_1",
	)
	if err != nil {
		t.Fatalf("show payment task: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\ttask_1",
		"Connector ID\tconn_1",
		"Status\tSUCCEEDED",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment task output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsTasksGetDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/tasks/task_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"task_1","status":"SUCCEEDED","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:01:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "tasks", "get", "task_1",
	)
	if err != nil {
		t.Fatalf("show payment task through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments tasks show") {
		t.Fatalf("expected deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsTransferInitiationListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"ti_1","reference":"ref","type":"TRANSFER","status":"PROCESSED","asset":"USD/2","amount":100,"connectorID":"conn_1","provider":"stripe","createdAt":"2026-01-01T00:00:00Z","scheduledAt":"2026-01-02T00:00:00Z","description":"desc","metadata":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list transfer initiations: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "ti_1\tTRANSFER\t100\tUSD/2\tPROCESSED\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected transfer initiation output:\n%s", stdout)
	}
}

func TestPaymentsTransferInitiationCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "transfer-initiation", "create", "--file", "request.json")
	if err == nil {
		t.Fatal("expected payments transfer-initiation create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments transfer-initiation create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsTransferInitiationCreateSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.URL.Query().Get("noValidation"); got != "true" {
				t.Fatalf("expected noValidation=true, got %q", got)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{
				`"amount":100`,
				`"asset":"USD/2"`,
				`"connectorID":"conn_1"`,
				`"sourceAccountID":"acc_src"`,
				`"destinationAccountID":"acc_dst"`,
				`"type":"TRANSFER"`,
			} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"paymentInitiationID":"ti_1","taskID":"task_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "create.json")
	if err := os.WriteFile(requestFile, []byte(`{"amount":100,"asset":"USD/2","connectorID":"conn_1","description":"desc","destinationAccountID":"acc_dst","metadata":{"env":"dev"},"reference":"ref","scheduledAt":"2026-01-02T00:00:00Z","sourceAccountID":"acc_src","type":"transfer","validated":true}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "create",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create transfer initiation: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Task ID: task_1",
		"Transfer initiation created with ID: ti_1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected create output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsTransferInitiationCreateDeprecatedPositionalFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"paymentInitiationID":"ti_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "create.json")
	if err := os.WriteFile(requestFile, []byte(`{"amount":"100","asset":"USD/2","connectorID":"conn_1","reference":"ref","scheduledAt":"2026-01-02T00:00:00Z","type":"PAYOUT"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "create", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create transfer initiation with positional file: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments transfer-initiation create --file <path>|-") {
		t.Fatalf("expected positional file deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsTransferInitiationShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations/ti_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"ti_1","reference":"ref","type":"TRANSFER","status":"PROCESSED","asset":"USD/2","amount":100,"connectorID":"conn_1","provider":"stripe","createdAt":"2026-01-01T00:00:00Z","scheduledAt":"2026-01-02T00:00:00Z","description":"desc","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "show", "ti_1",
	)
	if err != nil {
		t.Fatalf("show transfer initiation: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"ID\tti_1",
		"Reference\tref",
		"Amount\t100",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected transfer initiation output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsTransferInitiationDeprecatedAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		case "/api/payments/v3/payment-initiations/ti_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"ti_1","reference":"ref","type":"TRANSFER","status":"PROCESSED","asset":"USD/2","amount":100,"connectorID":"conn_1","provider":"stripe","createdAt":"2026-01-01T00:00:00Z","scheduledAt":"2026-01-02T00:00:00Z","description":"desc","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer_initiation", "list",
	)
	if err != nil {
		t.Fatalf("list transfer initiations through prefix alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments transfer-initiation") {
		t.Fatalf("expected prefix deprecation warning, got:\n%s", stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "get", "ti_1",
	)
	if err != nil {
		t.Fatalf("show transfer initiation through get alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments transfer-initiation show") {
		t.Fatalf("expected get deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsTransferInitiationApproveSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations/ti_1/approve":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"taskID":"task_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "approve", "ti_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("approve transfer initiation: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Task ID: task_1",
		"Transfer initiation ti_1 approved.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected approve output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsTransferInitiationRejectRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "transfer-initiation", "reject", "ti_1")
	if err == nil {
		t.Fatal("expected payments transfer-initiation reject to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments transfer-initiation reject requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsTransferInitiationRetrySelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations/ti_1/retry":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"taskID":"task_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "retry", "ti_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("retry transfer initiation: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Transfer initiation ti_1 queued for retry.") {
		t.Fatalf("unexpected retry output:\n%s", stdout)
	}
}

func TestPaymentsTransferInitiationDeleteSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations/ti_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "delete", "ti_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete transfer initiation: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Transfer initiation ti_1 deleted.") {
		t.Fatalf("unexpected delete output:\n%s", stdout)
	}
}

func TestPaymentsTransferInitiationReverseRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "transfer-initiation", "reverse", "ti_1", "--file", "request.json")
	if err == nil {
		t.Fatal("expected payments transfer-initiation reverse to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments transfer-initiation reverse requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsTransferInitiationReverseSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations/ti_1/reverse":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"amount":100`, `"asset":"USD/2"`, `"reference":"ref"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected reverse body to contain %q, got %s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"taskID":"task_1","paymentInitiationReversalID":"rev_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "reverse.json")
	if err := os.WriteFile(requestFile, []byte(`{"amount":100,"asset":"USD/2","description":"desc","metadata":{"env":"dev"},"reference":"ref"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "reverse", "ti_1",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("reverse transfer initiation: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Task ID: task_1",
		"Reversal ID: rev_1",
		"Transfer initiation ti_1 reversed.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected reverse output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsTransferInitiationReverseDeprecatedPositionalFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/payment-initiations/ti_1/reverse":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"taskID":"task_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	requestFile := filepath.Join(t.TempDir(), "reverse.json")
	if err := os.WriteFile(requestFile, []byte(`{"amount":"100","asset":"USD/2","reference":"ref"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "reverse", "ti_1", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("reverse transfer initiation with positional file: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments transfer-initiation reverse <transfer-initiation-id> --file <path>|-") {
		t.Fatalf("expected positional file deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsTransferInitiationUpdateStatusRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "transfer-initiation", "update-status", "ti_1", "VALIDATED")
	if err == nil {
		t.Fatal("expected payments transfer-initiation update-status to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments transfer-initiation update-status requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsTransferInitiationUpdateStatusSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/transfer-initiations/ti_1/status":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"status":"VALIDATED"`) {
				t.Fatalf("expected status body, got %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "update-status", "ti_1", "validated",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("update transfer initiation status: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Transfer initiation ti_1 status updated to VALIDATED.") {
		t.Fatalf("unexpected update-status output:\n%s", stdout)
	}
}

func TestPaymentsTransferInitiationUpdateStatusDeprecatedAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/transfer-initiations/ti_1/status":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "transfer-initiation", "update_status", "ti_1", "REJECTED",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("update transfer initiation status through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "use payments transfer-initiation update-status") {
		t.Fatalf("expected update_status deprecation warning, got:\n%s", stderr)
	}
}

func TestPaymentsConnectorsListSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/connectors":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"conn_1","name":"Stripe EU","provider":"stripe","reference":"ref","createdAt":"2026-01-01T00:00:00Z","scheduledForDeletion":false,"config":{}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "connectors", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list payment connectors: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"stripe\tStripe EU\tconn_1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment connectors output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsConnectorsUninstallRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "connectors", "uninstall", "conn_1")
	if err == nil {
		t.Fatal("expected payments connectors uninstall to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments connectors uninstall requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsConnectorsUninstallSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/connectors/conn_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":{"taskID":"task_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "connectors", "uninstall", "conn_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("uninstall payment connector: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Task ID: task_1",
		"Connector conn_1 uninstall scheduled.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment connector uninstall output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsConnectorsUninstallPinnedV1RequiresProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"1.9.0","health":true}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "connectors", "uninstall", "conn_1",
		"--api-version", "v1",
		"--confirm",
	)
	if err == nil {
		t.Fatal("expected v1 uninstall without provider to fail")
	}
	if !strings.Contains(err.Error(), "provider is required") {
		t.Fatalf("expected provider error, got: %v stderr=%s", err, stderr)
	}
}

func TestPaymentsConnectorsInstallRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "payments", "connectors", "install", "stripe", "--file", "request.json")
	if err == nil {
		t.Fatal("expected payments connectors install to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "payments connectors install requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestPaymentsConnectorsInstallSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/connectors/install/stripe":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"apiKey":"sk_test"`, `"name":"Stripe EU"`, `"provider":"Stripe"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected install body to contain %q, got:\n%s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"data":"conn_1"}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}
	requestFile := filepath.Join(t.TempDir(), "stripe.json")
	if err := os.WriteFile(requestFile, []byte(`{"apiKey":"sk_test","name":"Stripe EU"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "connectors", "install", "stripe",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("install payment connector: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Connector stripe installed with ID: conn_1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment connector install output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestPaymentsConnectorsConfigShowSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/connectors/conn_1/config":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"apiKey":"sk_test","name":"Stripe EU","provider":"Stripe"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "connectors", "config", "show", "conn_1",
	)
	if err != nil {
		t.Fatalf("show payment connector config: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Connector ID: conn_1",
		"Provider: stripe",
		`"name": "Stripe EU"`,
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment connector config output to contain %q, got:\n%s", expected, stdout)
		}
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"payments", "connectors", "get-config",
		"--connector-id", "conn_1",
	)
	if err != nil {
		t.Fatalf("show payment connector config through deprecated get-config: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command payments connectors get-config has been deprecated, use payments connectors config show <connector-id>") {
		t.Fatalf("expected get-config deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "API version: v3") || !strings.Contains(stdout, "Connector ID: conn_1") {
		t.Fatalf("unexpected deprecated get-config output:\n%s", stdout)
	}
}

func TestPaymentsConnectorsConfigUpdateSelectsV3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"payments","version":"3.1.0","health":true}]}`)
		case "/api/payments/v3/connectors/conn_1/config":
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"apiKey":"sk_test"`, `"name":"Stripe EU"`, `"provider":"Stripe"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected update body to contain %q, got:\n%s", expected, body)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}
	requestFile := filepath.Join(t.TempDir(), "stripe.json")
	if err := os.WriteFile(requestFile, []byte(`{"apiKey":"sk_test","name":"Stripe EU"}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"payments", "connectors", "config", "update", "conn_1",
		"--provider", "stripe",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("update payment connector config: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"API version: v3",
		"Connector conn_1 config updated.",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected payment connector update output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestWalletsCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "wallets", "create", "Main")
	if err == nil {
		t.Fatal("expected wallets create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "wallets create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestWalletsCreateSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "ik_1" {
				t.Fatalf("expected idempotency key ik_1, got %q", got)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"Main"`, `"env":"dev"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create wallet body to contain %q, got:\n%s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"wallet_1","name":"Main","ledger":"default","createdAt":"2026-01-01T00:00:00Z","metadata":{"env":"dev"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "create", "Main",
		"--metadata", "env=dev",
		"--idempotency-key", "ik_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create wallet: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Wallet created with ID: wallet_1") {
		t.Fatalf("unexpected create wallet output:\n%s", stdout)
	}
}

func TestWalletsListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"wallet_1","name":"Main","ledger":"default","createdAt":"2026-01-01T00:00:00Z","metadata":{"env":"dev"}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list wallets: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "wallet_1\tMain\tdefault\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected wallets list output:\n%s", stdout)
	}
}

func TestWalletsShowSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets/wallet_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"wallet_1","name":"Main","ledger":"default","createdAt":"2026-01-01T00:00:00Z","metadata":{"env":"dev"},"balances":{"main":{"assets":{}}}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "wallets", "show", "wallet_1")
	if err != nil {
		t.Fatalf("show wallet: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "ID\twallet_1", "Name\tMain", "Ledger\tdefault"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected wallet output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestWalletsUpdateSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets/wallet_1":
			if r.Method != http.MethodPatch {
				t.Fatalf("expected PATCH, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"env":"prod"`) {
				t.Fatalf("expected update wallet body to contain metadata, got:\n%s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "update", "wallet_1",
		"--metadata", "env=prod",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("update wallet: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Wallet wallet_1 updated.") {
		t.Fatalf("unexpected update wallet output:\n%s", stdout)
	}
}

func TestWalletsCreditRequiresExplicitWalletID(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "wallets", "credit", "--amount", "100", "--asset", "USD/2", "--confirm")
	if err == nil {
		t.Fatal("expected wallets credit to require wallet id")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("expected explicit wallet id error, got: %v", err)
	}
}

func TestWalletsCreditSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets/wallet_1/credit":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"amount":100`, `"asset":"USD/2"`, `"balance":"main"`, `"env":"dev"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected credit body to contain %q, got:\n%s", expected, body)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "credit", "wallet_1",
		"--amount", "100",
		"--asset", "USD/2",
		"--balance", "main",
		"--metadata", "env=dev",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("credit wallet: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Wallet wallet_1 credited.") {
		t.Fatalf("unexpected credit wallet output:\n%s", stdout)
	}
}

func TestWalletsDebitSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets/wallet_1/debit":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"amount":100`, `"asset":"USD/2"`, `"balances":["main"]`, `"pending":true`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected debit body to contain %q, got:\n%s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"hold_1","walletID":"wallet_1","asset":"USD/2","description":"test","metadata":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "debit", "wallet_1",
		"--amount", "100",
		"--asset", "USD/2",
		"--balance", "main",
		"--pending",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("debit wallet: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "Hold ID: hold_1", "Wallet wallet_1 debited."} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected debit wallet output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestWalletsBalancesCreateRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "wallets", "balances", "create", "wallet_1", "main")
	if err == nil {
		t.Fatal("expected wallets balances create to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "wallets balances create requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestWalletsBalancesCreateSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets/wallet_1/balances":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "ik_1" {
				t.Fatalf("expected idempotency key ik_1, got %q", got)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"name":"main"`, `"priority":10`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected create balance body to contain %q, got:\n%s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"name":"main","priority":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "balances", "create", "wallet_1", "main",
		"--priority", "10",
		"--idempotency-key", "ik_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create wallet balance: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Balance main created on wallet wallet_1.") {
		t.Fatalf("unexpected create wallet balance output:\n%s", stdout)
	}
}

func TestWalletsBalancesListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets/wallet_1/balances":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"name":"main","priority":10}],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "wallets", "balances", "list", "wallet_1")
	if err != nil {
		t.Fatalf("list wallet balances: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "main\t10") {
		t.Fatalf("unexpected list wallet balances output:\n%s", stdout)
	}
}

func TestWalletsBalancesShowSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/wallets/wallet_1/balances/main":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"name":"main","priority":10,"assets":{"USD/2":100}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "wallets", "balances", "show", "wallet_1", "main")
	if err != nil {
		t.Fatalf("show wallet balance: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "Name\tmain", "Priority\t10", "Asset\tUSD/2\t100"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected wallet balance output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestWalletsHoldsListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/holds":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if got := r.URL.Query().Get("walletID"); got != "wallet_1" {
				t.Fatalf("expected walletID wallet_1, got %q", got)
			}
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"hold_1","walletID":"wallet_1","asset":"USD/2","description":"test","metadata":{"env":"dev"}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "holds", "list",
		"--wallet-id", "wallet_1",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list wallet holds: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "hold_1\twallet_1\tUSD/2") {
		t.Fatalf("unexpected list wallet holds output:\n%s", stdout)
	}
}

func TestWalletsHoldsShowSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/holds/hold_1":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"hold_1","walletID":"wallet_1","asset":"USD/2","description":"test","originalAmount":100,"remaining":40,"metadata":{"env":"dev"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "wallets", "holds", "show", "hold_1")
	if err != nil {
		t.Fatalf("show wallet hold: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "ID\thold_1", "Wallet ID\twallet_1", "Asset\tUSD/2", "Remaining\t40"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected wallet hold output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestWalletsHoldsVoidRequiresConfirm(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "wallets", "holds", "void", "hold_1")
	if err == nil {
		t.Fatal("expected wallets holds void to require confirmation")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "wallets holds void requires --confirm") {
		t.Fatalf("expected confirmation error, got: %v", err)
	}
}

func TestWalletsHoldsVoidSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/holds/hold_1/void":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "ik_1" {
				t.Fatalf("expected idempotency key ik_1, got %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "holds", "void", "hold_1",
		"--idempotency-key", "ik_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("void wallet hold: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Hold hold_1 voided.") {
		t.Fatalf("unexpected void wallet hold output:\n%s", stdout)
	}
}

func TestWalletsHoldsConfirmSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/holds/hold_1/confirm":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"amount":40`, `"final":true`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected confirm hold body to contain %q, got:\n%s", expected, body)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "holds", "confirm", "hold_1",
		"--amount", "40",
		"--final",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("confirm wallet hold: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Hold hold_1 confirmed.") {
		t.Fatalf("unexpected confirm wallet hold output:\n%s", stdout)
	}
}

func TestWalletsTransactionsListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"wallets","version":"1.2.0","health":true}]}`)
		case "/api/wallets/transactions":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if got := r.URL.Query().Get("walletID"); got != "wallet_1" {
				t.Fatalf("expected walletID wallet_1, got %q", got)
			}
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":42,"ledger":"default","reference":"ref_1","timestamp":"2026-01-01T00:00:00Z","postings":[],"metadata":{"env":"dev"}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"wallets", "transactions", "list",
		"--wallet-id", "wallet_1",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list wallet transactions: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "42\t2026-01-01T00:00:00Z\tdefault\tref_1") {
		t.Fatalf("unexpected list wallet transactions output:\n%s", stdout)
	}
}

func TestFlowsWorkflowsListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/workflows":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"workflow_1","config":{"name":"Payout","stages":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"flows", "workflows", "list",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list workflows: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "workflow_1\tPayout\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected workflows list output:\n%s", stdout)
	}
}

func TestFlowsWorkflowsShowSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"1.5.0","health":true}]}`)
		case "/api/orchestration/workflows/workflow_1":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"workflow_1","config":{"name":"Payout","stages":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "workflows", "show", "workflow_1")
	if err != nil {
		t.Fatalf("show workflow: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "ID\tworkflow_1", "Name\tPayout", "Updated at\t2026-01-02T00:00:00Z"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected workflow output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestFlowsWorkflowsCreateSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/workflows":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"name":"Payout"`) {
				t.Fatalf("expected create workflow body to contain name, got:\n%s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"workflow_1","config":{"name":"Payout","stages":[]},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	requestFile := filepath.Join(t.TempDir(), "workflow.json")
	if err := os.WriteFile(requestFile, []byte(`{"name":"Payout","stages":[]}`), 0o600); err != nil {
		t.Fatalf("write workflow request: %v", err)
	}
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"flows", "workflows", "create",
		"--file", requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create workflow: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Workflow created with ID: workflow_1") {
		t.Fatalf("unexpected workflow create output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"flows", "workflows", "create",
		requestFile,
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create workflow with deprecated positional file: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Positional file has been deprecated, use flows workflows create --file <path>|-") {
		t.Fatalf("expected positional file warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Workflow created with ID: workflow_1") {
		t.Fatalf("unexpected deprecated workflow create output:\n%s", stdout)
	}
}

func TestFlowsWorkflowsDeleteSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/workflows/workflow_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"flows", "workflows", "delete", "workflow_1",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("delete workflow: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Workflow workflow_1 deleted.") {
		t.Fatalf("unexpected workflow delete output:\n%s", stdout)
	}
}

func TestFlowsWorkflowsRunSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/workflows/workflow_1/instances":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if got := r.URL.Query().Get("wait"); got != "true" {
				t.Fatalf("expected wait=true, got %q", got)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"env":"dev"`) {
				t.Fatalf("expected workflow run body to contain vars, got:\n%s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"instance_1","workflowID":"workflow_1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","terminated":false,"status":[]}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"flows", "workflows", "run", "workflow_1",
		"--variable", "env=dev",
		"--wait",
	)
	if err != nil {
		t.Fatalf("run workflow: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Workflow instance created with ID: instance_1") {
		t.Fatalf("unexpected workflow run output:\n%s", stdout)
	}
}

func TestFlowsInstancesListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/instances":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if got := r.URL.Query().Get("workflowID"); got != "workflow_1" {
				t.Fatalf("expected workflowID workflow_1, got %q", got)
			}
			if got := r.URL.Query().Get("running"); got != "true" {
				t.Fatalf("expected running true, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"instance_1","workflowID":"workflow_1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","terminated":false,"status":[]}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"flows", "instances", "list",
		"--workflow-id", "workflow_1",
		"--running",
		"--page-size", "10",
	)
	if err != nil {
		t.Fatalf("list flow instances: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "instance_1\tworkflow_1\tfalse\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected flow instances output:\n%s", stdout)
	}
}

func TestFlowsInstancesShowSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"1.5.0","health":true}]}`)
		case "/api/orchestration/instances/instance_1":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"instance_1","workflowID":"workflow_1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","terminated":false,"status":[]}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "instances", "show", "instance_1")
	if err != nil {
		t.Fatalf("show flow instance: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "ID\tinstance_1", "Workflow ID\tworkflow_1", "Terminated\tfalse"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected instance output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestFlowsInstancesSendEventSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/instances/instance_1/events":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"name":"approved"`) {
				t.Fatalf("expected send event body to contain event name, got:\n%s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "instances", "send-event", "instance_1", "approved")
	if err != nil {
		t.Fatalf("send flow instance event: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Event approved sent to instance instance_1.") {
		t.Fatalf("unexpected send event output:\n%s", stdout)
	}
}

func TestFlowsInstancesStopSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/instances/instance_1/abort":
			if r.Method != http.MethodPut {
				t.Fatalf("expected PUT, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "instances", "stop", "instance_1", "--confirm")
	if err != nil {
		t.Fatalf("stop flow instance: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Workflow instance instance_1 stopped.") {
		t.Fatalf("unexpected stop instance output:\n%s", stdout)
	}
}

func TestFlowsTriggersCreateSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/triggers":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			for _, expected := range []string{`"event":"approved"`, `"workflowID":"workflow_1"`, `"name":"Payout"`} {
				if !strings.Contains(body, expected) {
					t.Fatalf("expected trigger create body to contain %q, got:\n%s", expected, body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"trigger_1","name":"Payout","event":"approved","workflowID":"workflow_1","createdAt":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"flows", "triggers", "create", "approved", "workflow_1",
		"--name", "Payout",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("create flow trigger: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Trigger created with ID: trigger_1") {
		t.Fatalf("unexpected create trigger output:\n%s", stdout)
	}
}

func TestFlowsTriggersListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/triggers":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if got := r.URL.Query().Get("name"); got != "Payout" {
				t.Fatalf("expected name Payout, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"trigger_1","name":"Payout","event":"approved","workflowID":"workflow_1","createdAt":"2026-01-01T00:00:00Z"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "triggers", "list", "--name", "Payout")
	if err != nil {
		t.Fatalf("list flow triggers: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "trigger_1\tPayout\tapproved\tworkflow_1") {
		t.Fatalf("unexpected list triggers output:\n%s", stdout)
	}
}

func TestFlowsTriggersShowSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"1.5.0","health":true}]}`)
		case "/api/orchestration/triggers/trigger_1":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"trigger_1","name":"Payout","event":"approved","workflowID":"workflow_1","createdAt":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "triggers", "show", "trigger_1")
	if err != nil {
		t.Fatalf("show flow trigger: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{"API version: v1", "ID\ttrigger_1", "Name\tPayout", "Event\tapproved"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected trigger output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestFlowsTriggersDeleteSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/triggers/trigger_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "triggers", "delete", "trigger_1", "--confirm")
	if err != nil {
		t.Fatalf("delete flow trigger: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Trigger trigger_1 deleted.") {
		t.Fatalf("unexpected delete trigger output:\n%s", stdout)
	}
}

func TestFlowsTriggersTestSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/triggers/trigger_1/test":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			body := readRequestBody(t, r)
			if !strings.Contains(body, `"name":"approved"`) {
				t.Fatalf("expected trigger test body to contain event name, got:\n%s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"filter":{"match":true},"variables":{}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "triggers", "test", "trigger_1", `{"name":"approved"}`)
	if err != nil {
		t.Fatalf("test flow trigger: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "Filter match\ttrue") {
		t.Fatalf("unexpected trigger test output:\n%s", stdout)
	}
}

func TestFlowsTriggersOccurrencesListSelectsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/triggers/trigger_1/occurrences":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"triggerID":"trigger_1","workflowInstanceID":"instance_1","date":"2026-01-01T00:00:00Z","event":{"name":"approved"}}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "flows", "triggers", "occurrences", "list", "trigger_1", "--page-size", "10")
	if err != nil {
		t.Fatalf("list trigger occurrences: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v2") || !strings.Contains(stdout, "trigger_1\tinstance_1\t2026-01-01T00:00:00Z") {
		t.Fatalf("unexpected trigger occurrences output:\n%s", stdout)
	}
}

func TestOrchestrationAliasWarns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"orchestration","version":"2.1.0","health":true}]}`)
		case "/api/orchestration/v2/workflows":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[],"hasMore":false,"pageSize":15}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	_, stderr, err = executeCommand(t, "--config-dir", configDir, "orchestration", "workflows", "list")
	if err != nil {
		t.Fatalf("list workflows through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command orchestration has been deprecated, use flows") {
		t.Fatalf("expected orchestration alias warning, got:\n%s", stderr)
	}
}

func TestFlowsHelpUsesCanonicalProductName(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "flows", "workflows", "create", "--help")
	if err != nil {
		t.Fatalf("flows workflows create help: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Pin flows API version") {
		t.Fatalf("expected flows API version help, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "Pin orchestration API version") {
		t.Fatalf("help should not expose orchestration as the canonical product name:\n%s", stdout)
	}
}

func TestReconciliationListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"reconciliation","version":"1.0.0","health":true}]}`)
		case "/api/reconciliation/reconciliations":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			if got := r.URL.Query().Get("query"); !strings.Contains(got, "policy_1") || !strings.Contains(got, "COMPLETED") {
				t.Fatalf("expected policy and status query filters, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"rec_1","policyID":"policy_1","status":"COMPLETED","ledgerBalances":{},"paymentsBalances":{},"driftBalances":{},"reconciledAtLedger":"2026-01-01T00:00:00Z","reconciledAtPayments":"2026-01-01T00:00:00Z","createdAt":"2026-01-01T00:01:00Z"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "reconciliation", "list", "--page-size", "10", "--policy-id", "policy_1", "--status", "COMPLETED")
	if err != nil {
		t.Fatalf("list reconciliations: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "rec_1\tpolicy_1\tCOMPLETED\t2026-01-01T00:01:00Z") {
		t.Fatalf("unexpected reconciliations output:\n%s", stdout)
	}
}

func TestReconciliationShowGetWarns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"reconciliation","version":"1.0.0","health":true}]}`)
		case "/api/reconciliation/reconciliations/rec_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"rec_1","policyID":"policy_1","status":"FAILED","error":"drift detected","ledgerBalances":{},"paymentsBalances":{},"driftBalances":{},"reconciledAtLedger":"2026-01-01T00:00:00Z","reconciledAtPayments":"2026-01-01T00:00:00Z","createdAt":"2026-01-01T00:01:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "reconciliation", "get", "rec_1")
	if err != nil {
		t.Fatalf("show reconciliation through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command reconciliation get has been deprecated, use reconciliation show") {
		t.Fatalf("expected get deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "Status\tFAILED") || !strings.Contains(stdout, "Error\tdrift detected") {
		t.Fatalf("unexpected reconciliation output:\n%s", stdout)
	}
}

func TestReconciliationPoliciesListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"reconciliation","version":"1.0.0","health":true}]}`)
		case "/api/reconciliation/policies":
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Fatalf("expected pageSize 10, got %q", got)
			}
			if got := r.URL.Query().Get("query"); !strings.Contains(got, "default") || !strings.Contains(got, "pool_1") {
				t.Fatalf("expected ledger and pool query filters, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"policy_1","name":"daily","ledgerName":"default","paymentsPoolID":"pool_1","ledgerQuery":{},"createdAt":"2026-01-01T00:00:00Z"}],"hasMore":false,"pageSize":10}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "reconciliation", "policies", "list", "--page-size", "10", "--ledger", "default", "--payments-pool-id", "pool_1")
	if err != nil {
		t.Fatalf("list policies: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "policy_1\tdaily\tdefault\tpool_1") {
		t.Fatalf("unexpected policies output:\n%s", stdout)
	}
}

func TestReconciliationPoliciesShowGetWarns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"reconciliation","version":"1.0.0","health":true}]}`)
		case "/api/reconciliation/policies/policy_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"policy_1","name":"daily","ledgerName":"default","paymentsPoolID":"pool_1","ledgerQuery":{},"createdAt":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "reconciliation", "policies", "get", "policy_1")
	if err != nil {
		t.Fatalf("show policy through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command reconciliation policies get has been deprecated, use reconciliation policies show") {
		t.Fatalf("expected get deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "ID\tpolicy_1") || !strings.Contains(stdout, "Payments pool ID\tpool_1") {
		t.Fatalf("unexpected policy output:\n%s", stdout)
	}
}

func TestReconciliationPoliciesCreateSelectsV1(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"reconciliation","version":"1.0.0","health":true}]}`)
		case "/api/reconciliation/policies":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			requestBody = readRequestBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"policy_1","name":"daily","ledgerName":"default","paymentsPoolID":"pool_1","ledgerQuery":{},"createdAt":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	requestFile := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(requestFile, []byte(`{"name":"daily","ledgerName":"default","paymentsPoolID":"pool_1","ledgerQuery":{}}`), 0o600); err != nil {
		t.Fatalf("write request file: %v", err)
	}
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "reconciliation", "policies", "create", "--file", requestFile, "--confirm")
	if err != nil {
		t.Fatalf("create policy: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(requestBody, `"ledgerName":"default"`) || !strings.Contains(requestBody, `"paymentsPoolID":"pool_1"`) {
		t.Fatalf("unexpected policy request body: %s", requestBody)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Policy created with ID: policy_1") {
		t.Fatalf("unexpected create policy output:\n%s", stdout)
	}
}

func TestReconciliationPoliciesDeleteRequiresConfirm(t *testing.T) {
	_, _, err := executeCommand(t, "reconciliation", "policies", "delete", "policy_1")
	if err == nil || !strings.Contains(err.Error(), "requires --confirm") {
		t.Fatalf("expected confirm error, got %v", err)
	}
}

func TestReconciliationPoliciesDeleteSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"reconciliation","version":"1.0.0","health":true}]}`)
		case "/api/reconciliation/policies/policy_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "reconciliation", "policies", "delete", "policy_1", "--confirm")
	if err != nil {
		t.Fatalf("delete policy: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Policy policy_1 deleted.") {
		t.Fatalf("unexpected delete policy output:\n%s", stdout)
	}
}

func TestReconciliationPoliciesReconcileSelectsV1(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"reconciliation","version":"1.0.0","health":true}]}`)
		case "/api/reconciliation/policies/policy_1/reconciliation":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			requestBody = readRequestBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"rec_1","policyID":"policy_1","status":"PENDING","ledgerBalances":{},"paymentsBalances":{},"driftBalances":{},"reconciledAtLedger":"2026-01-01T00:00:00Z","reconciledAtPayments":"2026-01-01T00:05:00Z","createdAt":"2026-01-01T00:06:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"reconciliation", "policies", "reconcile", "policy_1",
		"--ledger-at", "2026-01-01T00:00:00Z",
		"--payments-at", "2026-01-01T00:05:00Z",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("reconcile policy: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(requestBody, `"reconciledAtLedger":"2026-01-01T00:00:00Z"`) || !strings.Contains(requestBody, `"reconciledAtPayments":"2026-01-01T00:05:00Z"`) {
		t.Fatalf("unexpected reconcile request body: %s", requestBody)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Reconciliation started with ID: rec_1") {
		t.Fatalf("unexpected reconcile output:\n%s", stdout)
	}

	requestBody = ""
	stdout, stderr, err = executeCommand(t,
		"--config-dir", configDir,
		"reconciliation", "policies", "reconcile", "policy_1",
		"2026-01-01T00:00:00Z",
		"2026-01-01T00:05:00Z",
		"--confirm",
	)
	if err != nil {
		t.Fatalf("reconcile policy with deprecated positionals: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Positional reconciliation timestamps have been deprecated, use --ledger-at and --payments-at") {
		t.Fatalf("expected positional timestamp deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(requestBody, `"reconciledAtLedger":"2026-01-01T00:00:00Z"`) || !strings.Contains(requestBody, `"reconciledAtPayments":"2026-01-01T00:05:00Z"`) {
		t.Fatalf("unexpected deprecated reconcile request body: %s", requestBody)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Reconciliation started with ID: rec_1") {
		t.Fatalf("unexpected deprecated reconcile output:\n%s", stdout)
	}
}

func TestAuthClientsListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"auth","version":"1.0.0","health":true}]}`)
		case "/api/auth/clients":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"id":"client_1","name":"backend","scopes":["ledger:read"],"redirectUris":["http://localhost/callback"],"secrets":[{"id":"secret_1","name":"default","lastDigits":"1234"}]}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "auth", "clients", "list")
	if err != nil {
		t.Fatalf("list auth clients: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "client_1\tbackend\tledger:read") {
		t.Fatalf("unexpected auth clients output:\n%s", stdout)
	}
}

func TestAuthClientsShowGetWarns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"auth","version":"1.0.0","health":true}]}`)
		case "/api/auth/clients/client_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"client_1","name":"backend","scopes":["ledger:read"],"redirectUris":["http://localhost/callback"],"secrets":[{"id":"secret_1","name":"default","lastDigits":"1234"}]}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "auth", "clients", "get", "client_1")
	if err != nil {
		t.Fatalf("show auth client through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command auth clients get has been deprecated, use auth clients show") {
		t.Fatalf("expected get deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "ID\tclient_1") || !strings.Contains(stdout, "Secrets\t1") {
		t.Fatalf("unexpected auth client output:\n%s", stdout)
	}
}

func TestAuthUsersListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"auth","version":"1.0.0","health":true}]}`)
		case "/api/auth/users":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"id":"user_1","email":"user@example.com","subject":"sub_1"}]}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "auth", "users", "list")
	if err != nil {
		t.Fatalf("list auth users: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "user_1\tuser@example.com\tsub_1") {
		t.Fatalf("unexpected auth users output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "auth", "clients", "users", "list")
	if err != nil {
		t.Fatalf("list auth users through clients alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command auth clients users has been deprecated, use auth users") {
		t.Fatalf("expected auth clients users deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "user_1\tuser@example.com\tsub_1") {
		t.Fatalf("unexpected auth clients users output:\n%s", stdout)
	}
}

func TestAuthUsersShowGetWarns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"auth","version":"1.0.0","health":true}]}`)
		case "/api/auth/users/user_1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"user_1","email":"user@example.com","subject":"sub_1"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "auth", "users", "get", "user_1")
	if err != nil {
		t.Fatalf("show auth user through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command auth users get has been deprecated, use auth users show") {
		t.Fatalf("expected get deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "ID\tuser_1") || !strings.Contains(stdout, "Email\tuser@example.com") {
		t.Fatalf("unexpected auth user output:\n%s", stdout)
	}
}

func TestAuthClientsCreateSelectsV1(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"auth","version":"1.0.0","health":true}]}`)
		case "/api/auth/clients":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			requestBody = readRequestBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"client_1","name":"backend","scopes":["ledger:read"],"redirectUris":["http://localhost/callback"],"trusted":true}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"auth", "clients", "create", "backend",
		"--scope", "ledger:read",
		"--redirect-uri", "http://localhost/callback",
		"--trusted",
	)
	if err != nil {
		t.Fatalf("create auth client: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{`"name":"backend"`, `"scopes":["ledger:read"]`, `"redirectUris":["http://localhost/callback"]`, `"trusted":true`} {
		if !strings.Contains(requestBody, expected) {
			t.Fatalf("expected request body to contain %s, got %s", expected, requestBody)
		}
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "Client client_1 created.") {
		t.Fatalf("unexpected create auth client output:\n%s", stdout)
	}
}

func TestAuthClientsDeleteRequiresConfirm(t *testing.T) {
	_, _, err := executeCommand(t, "auth", "clients", "delete", "client_1")
	if err == nil || !strings.Contains(err.Error(), "requires --confirm") {
		t.Fatalf("expected confirm error, got %v", err)
	}
}

func TestAuthClientsSecretsCreateDoesNotPrintClearSecretInPlainOutput(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"auth","version":"1.0.0","health":true}]}`)
		case "/api/auth/clients/client_1/secrets":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			requestBody = readRequestBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"secret_1","name":"default","lastDigits":"1234","clear":"super-secret"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "auth", "clients", "secrets", "create", "client_1", "default")
	if err != nil {
		t.Fatalf("create auth secret: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(requestBody, `"name":"default"`) {
		t.Fatalf("unexpected secret request body: %s", requestBody)
	}
	if strings.Contains(stdout, "super-secret") {
		t.Fatalf("plain output must not include clear secret:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Secret default created for client client_1") {
		t.Fatalf("unexpected create secret output:\n%s", stdout)
	}
}

func TestWebhooksListSelectsV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"webhooks","version":"1.0.0","health":true}]}`)
		case "/api/webhooks/configs":
			if got := r.URL.Query().Get("endpoint"); got != "https://example.com/webhook" {
				t.Fatalf("expected endpoint filter, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"cursor":{"data":[{"id":"wh_1","endpoint":"https://example.com/webhook","eventTypes":["ledger.transaction.created"],"active":true,"secret":"super-secret","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],"hasMore":false}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "webhooks", "list", "--endpoint", "https://example.com/webhook")
	if err != nil {
		t.Fatalf("list webhooks: %v stderr=%s", err, stderr)
	}
	if strings.Contains(stdout, "super-secret") {
		t.Fatalf("plain output must not include webhook secret:\n%s", stdout)
	}
	if !strings.Contains(stdout, "API version: v1") || !strings.Contains(stdout, "wh_1\thttps://example.com/webhook\ttrue\tledger.transaction.created") {
		t.Fatalf("unexpected webhooks output:\n%s", stdout)
	}
}

func TestWebhooksCreateSelectsV1AndMasksSecret(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"webhooks","version":"1.0.0","health":true}]}`)
		case "/api/webhooks/configs":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			requestBody = readRequestBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"wh_1","endpoint":"https://example.com/webhook","eventTypes":["ledger.transaction.created"],"active":false,"secret":"super-secret","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"webhooks", "create", "https://example.com/webhook", "ledger.transaction.created",
		"--secret", "super-secret",
	)
	if err != nil {
		t.Fatalf("create webhook: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{`"endpoint":"https://example.com/webhook"`, `"eventTypes":["ledger.transaction.created"]`, `"secret":"super-secret"`} {
		if !strings.Contains(requestBody, expected) {
			t.Fatalf("expected request body to contain %s, got %s", expected, requestBody)
		}
	}
	if strings.Contains(stdout, "super-secret") {
		t.Fatalf("plain output must not include webhook secret:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Webhook config wh_1 created.") {
		t.Fatalf("unexpected create webhook output:\n%s", stdout)
	}
}

func TestWebhooksDeleteRequiresConfirm(t *testing.T) {
	_, _, err := executeCommand(t, "webhooks", "delete", "wh_1")
	if err == nil || !strings.Contains(err.Error(), "requires --confirm") {
		t.Fatalf("expected confirm error, got %v", err)
	}
}

func TestWebhooksActionsSelectV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"webhooks","version":"1.0.0","health":true}]}`)
		case "/api/webhooks/configs/wh_1/activate":
			if r.Method != http.MethodPut {
				t.Fatalf("expected PUT activate, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"wh_1","endpoint":"https://example.com/webhook","eventTypes":["ledger.transaction.created"],"active":true,"secret":"secret","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:01:00Z"}}`)
		case "/api/webhooks/configs/wh_1/deactivate":
			if r.Method != http.MethodPut {
				t.Fatalf("expected PUT deactivate, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"wh_1","endpoint":"https://example.com/webhook","eventTypes":["ledger.transaction.created"],"active":false,"secret":"secret","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:02:00Z"}}`)
		case "/api/webhooks/configs/wh_1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "webhooks", "activate", "wh_1")
	if err != nil {
		t.Fatalf("activate webhook: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Webhook config wh_1 activated.") {
		t.Fatalf("unexpected activate output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "webhooks", "deactivate", "wh_1", "--confirm")
	if err != nil {
		t.Fatalf("deactivate webhook: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Webhook config wh_1 deactivated.") {
		t.Fatalf("unexpected deactivate output:\n%s", stdout)
	}

	stdout, stderr, err = executeCommand(t, "--config-dir", configDir, "webhooks", "delete", "wh_1", "--confirm")
	if err != nil {
		t.Fatalf("delete webhook: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Webhook config wh_1 deleted.") {
		t.Fatalf("unexpected delete output:\n%s", stdout)
	}
}

func TestWebhooksChangeSecretAliasWarns(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"name":"webhooks","version":"1.0.0","health":true}]}`)
		case "/api/webhooks/configs/wh_1/secret/change":
			if r.Method != http.MethodPut {
				t.Fatalf("expected PUT, got %s", r.Method)
			}
			requestBody = readRequestBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{"id":"wh_1","endpoint":"https://example.com/webhook","eventTypes":["ledger.transaction.created"],"active":true,"secret":"rotated-secret","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:01:00Z"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	_, stderr, err := executeCommand(t,
		"--config-dir", configDir,
		"context", "create", "stack", "local",
		"--stack-url", server.URL,
	)
	if err != nil {
		t.Fatalf("create context: %v stderr=%s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "webhooks", "change-secret", "wh_1", "--secret", "rotated-secret")
	if err != nil {
		t.Fatalf("rotate webhook secret through alias: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stderr, "Command webhooks change-secret has been deprecated, use webhooks secret rotate") {
		t.Fatalf("expected change-secret deprecation warning, got:\n%s", stderr)
	}
	if !strings.Contains(requestBody, `"secret":"rotated-secret"`) {
		t.Fatalf("unexpected change secret request body: %s", requestBody)
	}
	if strings.Contains(stdout, "rotated-secret") {
		t.Fatalf("plain output must not include webhook secret:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Webhook config wh_1 secret rotated.") {
		t.Fatalf("unexpected change secret output:\n%s", stdout)
	}
}

func TestConfigMigrateV3DryRun(t *testing.T) {
	v3Dir := writeV3CommandFixture(t, true)

	stdout, stderr, err := executeCommand(t, "config", "migrate-v3", "--from", v3Dir, "--dry-run")
	if err != nil {
		t.Fatalf("migrate dry-run: %v stderr=%s", err, stderr)
	}
	for _, expected := range []string{
		"Current context: default",
		"- default (cloud-stack)",
		"Credential moves: 1",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected migration output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestConfigMigrateV3Write(t *testing.T) {
	v3Dir := writeV3CommandFixture(t, false)
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "config", "migrate-v3", "--from", v3Dir)
	if err != nil {
		t.Fatalf("migrate write: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Migrated 1 context(s)") {
		t.Fatalf("unexpected migration output: %q", stdout)
	}

	cfg, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		t.Fatalf("load migrated config: %v", err)
	}
	if cfg.CurrentContext != "default" || cfg.Contexts["default"].Kind != v4config.ContextKindCloudStack {
		t.Fatalf("unexpected migrated config: %#v", cfg)
	}
}

func TestConfigMigrateV3UsesDefaultSourceDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	v3Dir := filepath.Join(homeDir, ".config", "formance", "fctl")
	writeV3CommandFixtureInDir(t, v3Dir, false)
	configDir := t.TempDir()

	stdout, stderr, err := executeCommand(t, "--config-dir", configDir, "config", "migrate-v3")
	if err != nil {
		t.Fatalf("migrate write from default v3 dir: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Migrated 1 context(s)") {
		t.Fatalf("unexpected migration output: %q", stdout)
	}
	if _, err := v4config.LoadFile(filepath.Join(configDir, "config.yaml")); err != nil {
		t.Fatalf("load migrated config: %v", err)
	}
}

func TestMissingConfigErrorsAreActionable(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "missing")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "session", args: []string{"session", "status"}},
		{name: "runtime", args: []string{"target", "inspect"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, err := executeCommand(t, append([]string{"--config-dir", configDir}, tc.args...)...)
			if err == nil {
				t.Fatal("expected missing config error")
			}
			for _, expected := range []string{
				"v4 config not found",
				filepath.Join(configDir, "config.yaml"),
				"fctl login",
				"fctl profile create stack",
				"fctl profile create cloud",
				"fctl profile create cloud-stack",
				"fctl config migrate-v3",
			} {
				if !strings.Contains(err.Error(), expected) {
					t.Fatalf("expected missing config error to contain %q, got %v stderr=%s", expected, err, stderr)
				}
			}
		})
	}
}

func writeV3CommandFixture(t *testing.T, withTokens bool) string {
	t.Helper()
	dir := t.TempDir()
	writeV3CommandFixtureInDir(t, dir, withTokens)
	return dir
}

func stringSliceContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func writeV3CommandFixtureInDir(t *testing.T, dir string, withTokens bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "profiles", "default"), 0o700); err != nil {
		t.Fatalf("create v3 fixture dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(`{"currentProfile":"default"}`), 0o600); err != nil {
		t.Fatalf("write v3 config: %v", err)
	}
	rootTokens := "null"
	if withTokens {
		rootTokens = `{"access":{"token":"access-token"},"id":{"token":"id-token"}}`
	}
	profile := `{
	  "membershipURI": "https://app.formance.cloud/api",
	  "rootTokens": ` + rootTokens + `,
	  "defaultOrganization": "org_123",
	  "defaultStack": "stack_123"
	}`
	if err := os.WriteFile(filepath.Join(dir, "profiles", "default", "profile.json"), []byte(profile), 0o600); err != nil {
		t.Fatalf("write v3 profile: %v", err)
	}
}
