package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIExecution(t *testing.T) {
	// Build the real entrypoint once: go run would hide the binary's exit status.
	binary := filepath.Join(t.TempDir(), "fctl")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "..") // #nosec G204 -- Fixed build command and test-owned output path.
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)

	for _, tt := range []struct {
		name        string
		args        []string
		withoutHome bool
		apiError    bool
		wantStatus  int
		wantOut     string
		wantErr     string
	}{
		{name: "help", args: []string{"--help"}, wantOut: "Usage:"},
		{name: "version", args: []string{"--version"}, wantOut: "develop"},
		{name: "json version", args: []string{"version", "-o", "json"}},
		{name: "unknown command", args: []string{"unknown-command"}, wantStatus: 255, wantErr: `unknown command "unknown-command"`},
		{name: "invalid arguments", args: []string{"version", "unexpected", "-o", "json"}, wantStatus: 255, wantErr: "accepts 0 arg(s), received 1"},
		{name: "missing profile", args: []string{"profiles", "show", "missing", "-o", "json"}, wantStatus: 255, wantErr: "profile.json"},
		{name: "not authenticated", args: []string{"stack", "list", "-o", "json"}, wantStatus: 255, wantErr: "Your authentication is invalid"},
		{name: "recovered panic", args: []string{"version"}, withoutHome: true, wantStatus: 255, wantErr: "is not defined"},
		{name: "non-JSON API error", args: []string{"cloud", "organizations", "list", "-o", "json"}, apiError: true, wantStatus: 2, wantErr: "synthetic upstream failure"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			configDir := filepath.Join(home, "config")
			require.NoError(t, os.Mkdir(configDir, 0700))
			if tt.apiError {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet || r.URL.Path != "/organizations" || r.Header.Get("Authorization") != "Bearer synthetic-token" {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					}
					http.Error(w, "synthetic upstream failure", http.StatusBadRequest)
				}))
				t.Cleanup(server.Close)
				profileDir := filepath.Join(configDir, "profiles", "default")
				require.NoError(t, os.MkdirAll(profileDir, 0700))
				profile := `{"membershipURI":"` + server.URL + `","rootTokens":{"accessToken":{"token":"synthetic-token","claims":{"exp":4102444800}}}}`
				require.NoError(t, os.WriteFile(filepath.Join(profileDir, "profile.json"), []byte(profile), 0600))
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, append([]string{"--config-dir", configDir}, tt.args...)...) // #nosec G204 -- Test-built binary, fixed cases and temporary config.
			// Do not inherit user profiles, credentials, proxies, or flag overrides.
			cmd.Env = []string{"XDG_CONFIG_HOME=" + home, "NO_COLOR=1", "TERM=dumb"}
			if !tt.withoutHome {
				cmd.Env = append(cmd.Env, "HOME="+home, "USERPROFILE="+home)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			require.NoError(t, ctx.Err(), "CLI timed out: %s", stderr.String())
			if tt.wantStatus == 0 {
				require.NoError(t, err, "%s", stderr.String())
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr, "stderr: %s", stderr.String())
				require.Equal(t, tt.wantStatus, exitErr.ExitCode())
			}
			if tt.wantErr != "" {
				require.Contains(t, stderr.String(), tt.wantErr)
				require.Empty(t, stdout.String(), "errors must not pollute stdout")
			} else {
				require.Empty(t, stderr.String())
				if tt.wantOut != "" {
					require.Contains(t, stdout.String(), tt.wantOut)
				} else {
					var document map[string]map[string]string
					require.NoError(t, json.Unmarshal(stdout.Bytes(), &document), "stdout: %q", stdout.String())
					require.Equal(t, map[string]map[string]string{
						"data": {"version": "develop", "commit": "-", "buildDate": "-"},
					}, document)
				}
			}
		})
	}
}
