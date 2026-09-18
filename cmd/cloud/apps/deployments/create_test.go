package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

// TestNewCreate_RequiredFlags asserts the cobra command exposes the flags
// required by the new {appId, manifestId, manifestVersion} payload shape.
// Until d063441 landed on the server, the deploy endpoint accepted inline
// YAML and an optional manifest version. The new shape requires a
// manifest version >= 1 — the validation happens at Run() time, but the
// flag must at least exist on the command tree.
func TestNewCreate_RequiredFlags(t *testing.T) {
	cmd := NewCreate()
	for _, name := range []string{"app-id", "manifest-id", "manifest-version", "wait", "wait-timeout"} {
		require.NotNil(t, cmd.Flags().Lookup(name), "flag %q must be defined", name)
	}
}

func TestNewCreate_WaitTimeoutDefault(t *testing.T) {
	cmd := NewCreate()
	f := cmd.Flags().Lookup("wait-timeout")
	require.NotNil(t, f)
	require.Equal(t, "30m", f.DefValue,
		"a sensible default keeps the previous unbounded-poll behaviour from coming back")
}

// Status constants must stay in sync with the server's deployment status
// enum (terraform-hcp-proxy `internal/storage/models/deployment.go`).
// Renaming any of these on the server without lifting the constants here
// would silently break the wait loop — these assertions are the canary.
func TestStatusConstants(t *testing.T) {
	require.Equal(t, "applied", statusApplied)
	require.Equal(t, "planned_and_finished", statusPlannedAndFinished)
	require.Equal(t, "errored", statusErrored)
}

func TestCreateWaitErrored(t *testing.T) {
	for _, output := range []string{"json", "plain"} {
		t.Run(output, func(t *testing.T) {
			cmd, stdout, _ := newTestCreateCommand(t, NewCreateCtrl(), deploymentAPI(t, []string{"errored"}, `{"data":[{"message":"apply failed","timestamp":"2026-01-01T00:00:00Z","module":"test","diagnostic":{"severity":"error","summary":"Invalid resource","detail":"Synthetic failure"}}]}`))
			require.NoError(t, cmd.Flags().Set("output", output))
			err := cmd.Execute()
			require.EqualError(t, err, "deployment failed: deployment-test\nerror: Invalid resource\nSynthetic failure")
			require.Empty(t, stdout.String(), "a failed deployment must not emit a success payload")
		})
	}
}

func TestCreateWaitSuccess(t *testing.T) {
	for _, status := range []string{"applied", "planned_and_finished"} {
		t.Run(status, func(t *testing.T) {
			ctrl := NewCreateCtrl()
			var delays []time.Duration
			ctrl.wait = func(ctx context.Context, delay time.Duration) error {
				delays = append(delays, delay)
				return ctx.Err()
			}
			cmd, stdout, stderr := newTestCreateCommand(t, ctrl, deploymentAPI(t,
				[]string{"pending", "planning", "planned", "applying", "applying", status}, ""))
			require.NoError(t, cmd.Execute())
			require.Equal(t, status, ctrl.store.Status)
			require.Equal(t, []time.Duration{0, 2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second, 15 * time.Second}, delays)
			var output struct {
				Data struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
			require.Equal(t, "deployment-test", output.Data.ID)
			require.Equal(t, status, output.Data.Status)
			require.Empty(t, stderr.String())
		})
	}
}

func TestCreateWaitFailureDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name string
		logs string
		want string
	}{
		{"empty logs", `{"data":[]}`, "deployment failed: deployment-test"},
		{"message without diagnostic", `{"data":[{"message":"apply failed","timestamp":"2026-01-01T00:00:00Z","module":"test"}]}`, "deployment failed: deployment-test\napply failed"},
		{"logs unavailable", "", "deployment failed: deployment-test (could not read logs:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, stdout, _ := newTestCreateCommand(t, NewCreateCtrl(), deploymentAPI(t, []string{"errored"}, tc.logs))
			require.ErrorContains(t, cmd.Execute(), tc.want)
			require.Empty(t, stdout.String())
		})
	}
}

func TestCreateWaitInterrupted(t *testing.T) {
	for _, interruption := range []error{context.DeadlineExceeded, context.Canceled} {
		t.Run(interruption.Error(), func(t *testing.T) {
			ctrl := NewCreateCtrl()
			ctrl.wait = func(ctx context.Context, delay time.Duration) error {
				if delay != 0 {
					return interruption
				}
				return ctx.Err()
			}
			cmd, stdout, _ := newTestCreateCommand(t, ctrl, deploymentAPI(t, []string{"applying"}, ""))
			err := cmd.Execute()
			require.ErrorIs(t, err, interruption)
			if interruption == context.DeadlineExceeded {
				require.ErrorContains(t, err, "timed out after 30m0s waiting for deployment deployment-test (last status: applying)")
			}
			require.Empty(t, stdout.String())
		})
	}
}

func TestCreateWaitExpiredDeadline(t *testing.T) {
	cmd, stdout, _ := newTestCreateCommand(t, NewCreateCtrl(), deploymentAPI(t, nil, ""))
	require.NoError(t, cmd.Flags().Set("wait-timeout", "0s"))
	err := cmd.Execute()
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "timed out after 0s waiting for deployment deployment-test (last status: pending)")
	require.Empty(t, stdout.String())
}

func TestCreateWaitCanceledDuringRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := deploymentAPI(t, nil, "")
	cmd, stdout, _ := newTestCreateCommand(t, NewCreateCtrl(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/deployments/deployment-test" {
			cancel()
			<-r.Context().Done()
			return
		}
		api.ServeHTTP(w, r)
	}))
	require.ErrorIs(t, cmd.ExecuteContext(ctx), context.Canceled)
	require.Empty(t, stdout.String())
}

func TestCreateWithoutWait(t *testing.T) {
	for _, output := range []string{"json", "plain"} {
		for _, status := range []string{"pending", "errored"} {
			t.Run(output+"/"+status, func(t *testing.T) {
				ctrl := NewCreateCtrl()
				ctrl.wait = func(context.Context, time.Duration) error {
					t.Fatal("--wait=false must not poll")
					return nil
				}
				cmd, stdout, _ := newTestCreateCommand(t, ctrl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.URL.Path != "/deployments" {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					fmt.Fprintf(w, `{"data":{"id":"deployment-test","appId":"app-test","status":%q}}`, status)
				}))
				require.NoError(t, cmd.Flags().Set("output", output))
				require.NoError(t, cmd.Flags().Set("wait", "false"))
				err := cmd.Execute()
				if status == "errored" {
					require.EqualError(t, err, "deployment failed: deployment-test")
					require.Empty(t, stdout.String())
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, status, ctrl.store.Status)
			})
		}
	}
}

func TestWaitForDeploymentPoll(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, waitForDeploymentPoll(ctx, time.Hour), context.Canceled)
	require.NoError(t, waitForDeploymentPoll(context.Background(), 0))
}

func deploymentAPI(t *testing.T, statuses []string, logs string) http.Handler {
	t.Helper()
	var reads atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /deployments":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"data":{"id":"deployment-test","appId":"app-test","status":"pending"}}`)
		case "GET /deployments/deployment-test":
			index := int(reads.Add(1)) - 1
			if index >= len(statuses) {
				t.Errorf("unexpected deployment poll %d", index+1)
				http.NotFound(w, r)
				return
			}
			fmt.Fprintf(w, `{"data":{"id":"deployment-test","appId":"app-test","status":%q}}`, statuses[index])
		case "GET /deployments/deployment-test/logs":
			if logs == "" {
				http.Error(w, `{"errorCode":"FORBIDDEN","errorMessage":"logs unavailable"}`, http.StatusForbidden)
				return
			}
			fmt.Fprintln(w, logs)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	})
}

func newTestCreateCommand(t *testing.T, ctrl *CreateCtrl, handler http.Handler) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cmd := newCreate(ctrl)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.Flags().String(fctl.ConfigDir, t.TempDir(), "")
	cmd.Flags().String(fctl.ProfileFlag, "test", "")
	cmd.Flags().String(fctl.OrganizationFlag, "organization-test", "")
	cmd.Flags().String(fctl.DeployAppAliasFlag, "deploy", "")
	cmd.Flags().StringP(fctl.OutputFlag, "o", "json", "")
	cmd.Flags().Bool(fctl.DebugFlag, false, "")
	cmd.Flags().Bool(fctl.InsecureTlsFlag, false, "")
	cmd.Flags().Bool(fctl.HTTPCloseOnErrorFlag, false, "")
	cmd.Flags().Bool(fctl.TelemetryFlag, false, "")
	cmd.SetArgs([]string{"--app-id=app-test", "--manifest-id=manifest-test", "--manifest-version=1"})
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	profile := fctl.Profile{MembershipURI: server.URL, RootTokens: &fctl.Tokens{}}
	profile.RootTokens.ID.Claims.Organizations = []fctl.OrganizationAccess{{
		ID: "organization-test", Applications: []fctl.ApplicationAccess{{Alias: "deploy"}},
	}}
	require.NoError(t, fctl.WriteProfile(cmd, "test", profile))
	var token fctl.AccessToken
	require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(`{"token":"synthetic-token","claims":{"exp":4102444800,"aud":[%q],"organization_id":"organization-test"}}`, server.URL)), &token))
	require.NoError(t, fctl.WriteAppToken(cmd, "test", "deploy", token))
	return cmd, stdout, stderr
}
