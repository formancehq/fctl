package stack

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v4/oidc"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestDisableCommandApproval(t *testing.T) {
	const stackPath = "/organizations/organization-test/stacks/stack-target"
	const listPath = "/organizations/organization-test/stacks"
	for _, tc := range []struct {
		name         string
		args         []string
		answer       string
		readResponse string
		wantError    string
		wantRefusal  bool
		wantPrompts  int
		wantRequests []string
	}{
		{
			name: "refusing an ID never mutates", args: []string{"stack-target"}, answer: "n",
			wantRefusal: true, wantPrompts: 1, wantRequests: []string{"GET " + stackPath},
		},
		{
			name: "refusing a name never mutates", args: []string{"--name=sandbox"}, answer: "n",
			wantRefusal: true, wantPrompts: 1, wantRequests: []string{"GET " + listPath},
		},
		{
			name: "confirm disables exactly the requested ID", args: []string{"stack-target", "--confirm"},
			wantRequests: []string{"GET " + stackPath, "PUT " + stackPath + "/disable"},
		},
		{
			name: "confirm disables exactly the named stack", args: []string{"--name=sandbox", "--confirm"},
			wantRequests: []string{"GET " + listPath, "PUT " + stackPath + "/disable"},
		},
		{
			name: "interactive approval disables once", args: []string{"stack-target"}, answer: "Y",
			wantPrompts: 1, wantRequests: []string{"GET " + stackPath, "PUT " + stackPath + "/disable"},
		},
		{
			name: "explicit false still requires approval", args: []string{"stack-target", "--confirm=false"}, answer: "n",
			wantRefusal: true, wantPrompts: 1, wantRequests: []string{"GET " + stackPath},
		},
		{
			name: "confirm cannot bypass a missing selector", args: []string{"--confirm"},
			wantError: "need either",
		},
		{
			name: "confirm cannot bypass conflicting selectors", args: []string{"stack-target", "--name=sandbox", "--confirm"},
			wantError: "need either",
		},
		{
			name: "confirm cannot bypass extra arguments", args: []string{"stack-target", "another-stack", "--confirm"},
			wantError: "accepts at most 1 arg",
		},
		{
			name: "confirm cannot bypass an empty ID", args: []string{"", "--confirm"},
			wantError: "need either",
		},
		{
			name: "unknown ID never mutates", args: []string{"unknown", "--confirm"},
			wantError: "404", wantRequests: []string{"GET /organizations/organization-test/stacks/unknown"},
		},
		{
			name: "unknown name never mutates", args: []string{"--name=unknown", "--confirm"},
			wantError: "Stack not found", wantRequests: []string{"GET " + listPath},
		},
		{
			name: "missing response ID never mutates", args: []string{"stack-target", "--confirm"},
			readResponse: `{"data":{"name":"sandbox"}}`, wantError: "missing stack id",
			wantRequests: []string{"GET " + stackPath},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests = append(requests, r.Method+" "+r.URL.RequestURI())
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPut:
					// Record every mutation, including one to an unexpected resource.
					w.WriteHeader(http.StatusAccepted)
				case r.Method == http.MethodGet && r.URL.Path == stackPath:
					response := tc.readResponse
					if response == "" {
						response = `{"data":{"id":"stack-target","name":"sandbox"}}`
					}
					_, _ = io.WriteString(w, response)
				case r.Method == http.MethodGet && r.URL.Path == listPath:
					_, _ = io.WriteString(w, `{"data":[{"id":"stack-other","name":"other"},{"id":"stack-target","name":"sandbox"}]}`)
				default:
					http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)

			cmd := newDisableTestCommand(t, server.URL)
			prompts := 0
			cmd.SetContext(fctl.WithApprovalPrompt(context.Background(), func(text string) (string, error) {
				prompts++
				require.Contains(t, text, "You are about to disable stack 'sandbox'")
				return tc.answer, nil
			}))
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			require.Equal(t, tc.wantPrompts, prompts)
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, tc.wantRequests, requests)
			switch {
			case tc.wantRefusal:
				require.ErrorIs(t, err, fctl.ErrMissingApproval)
			case tc.wantError != "":
				require.ErrorContains(t, err, tc.wantError)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func newDisableTestCommand(t *testing.T, serverURL string) *cobra.Command {
	t.Helper()
	t.Setenv("NAME", "")
	cmd := NewDisableCommand()
	cmd.Flags().String(fctl.ConfigDir, t.TempDir(), "")
	cmd.Flags().String(fctl.ProfileFlag, "approval-test", "")
	cmd.Flags().String(fctl.OutputFlag, "json", "")
	for _, flag := range []string{fctl.DebugFlag, fctl.InsecureTlsFlag, fctl.HTTPCloseOnErrorFlag, fctl.TelemetryFlag} {
		cmd.Flags().Bool(flag, false, "")
	}
	require.NoError(t, cmd.PersistentFlags().Set(fctl.OrganizationFlag, "organization-test"))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	require.NoError(t, fctl.WriteProfile(cmd, "approval-test", fctl.Profile{
		MembershipURI: serverURL,
		RootTokens: &fctl.Tokens{ID: fctl.IDToken{Claims: fctl.IDTokenClaims{
			Organizations: []fctl.OrganizationAccess{{ID: "organization-test"}},
		}}},
	}))
	require.NoError(t, fctl.WriteOrganizationToken(cmd, "approval-test", fctl.AccessToken{
		TokenWithClaims: fctl.TokenWithClaims[fctl.AccessTokenClaims]{
			Token: "synthetic-organization-token",
			Claims: fctl.AccessTokenClaims{
				TokenClaims:    oidc.TokenClaims{Expiration: oidc.Time(time.Now().Add(time.Hour).Unix())},
				OrganizationID: "organization-test",
			},
		},
	}))
	return cmd
}
