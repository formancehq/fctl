package internal

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v4/oidc"
	oidcclient "github.com/formancehq/go-libs/v4/oidc/client"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

func TestClientFactoryNonInteractiveModeRejectsUnusableTokensBeforeAuthentication(t *testing.T) {
	tests := map[string]*fctl.AccessToken{
		"missing": nil,
		"expired": accessTokenExpiringAt(time.Now().Add(-time.Minute)),
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			var terminal, commandOutput bytes.Buffer
			originalLogger := pterm.DefaultLogger
			pterm.DefaultLogger.Writer = &terminal
			t.Cleanup(func() { pterm.DefaultLogger = originalLogger })

			authenticationStarted := false
			factory := newClientFactory(clientFactoryDependencies{
				loadAndAuthenticateCurrentProfile: testProfileLoader(),
				resolveStackID: func(*cobra.Command, fctl.Profile) (string, string, error) {
					return "org", "stack", nil
				},
				readStackToken: func(*cobra.Command, string, string, string) (*fctl.AccessToken, error) {
					return token, nil
				},
				newStackClientsFromFlags: func(*cobra.Command, oidcclient.RelyingParty, fctl.Dialog, string, fctl.Profile) (*fctl.StackClients, error) {
					authenticationStarted = true
					return nil, nil
				},
			})
			command := &cobra.Command{}
			command.SetContext(WithNonInteractive(context.Background()))
			command.SetOut(&commandOutput)
			command.SetErr(&commandOutput)

			client, err := factory(command)
			if err == nil {
				t.Fatal("factory error = nil, want unusable-token error")
			}
			if client != nil {
				t.Fatalf("client = %#v, want nil", client)
			}
			if authenticationStarted {
				t.Fatal("interactive authentication boundary was called")
			}
			if terminal.Len() != 0 || commandOutput.Len() != 0 {
				t.Fatalf("completion emitted output: terminal=%q command=%q", terminal.String(), commandOutput.String())
			}
		})
	}
}

func TestClientFactoryNonInteractiveModeUsesSilentDialogWithValidToken(t *testing.T) {
	var terminal bytes.Buffer
	originalLogger := pterm.DefaultLogger
	pterm.DefaultLogger.Writer = &terminal
	t.Cleanup(func() { pterm.DefaultLogger = originalLogger })

	factory := newClientFactory(clientFactoryDependencies{
		loadAndAuthenticateCurrentProfile: testProfileLoader(),
		resolveStackID: func(*cobra.Command, fctl.Profile) (string, string, error) {
			return "org", "stack", nil
		},
		readStackToken: func(*cobra.Command, string, string, string) (*fctl.AccessToken, error) {
			return accessTokenExpiringAt(time.Now().Add(time.Hour)), nil
		},
		newStackClientsFromFlags: func(_ *cobra.Command, _ oidcclient.RelyingParty, dialog fctl.Dialog, _ string, _ fctl.Profile) (*fctl.StackClients, error) {
			dialog.Info("must stay silent")
			return &fctl.StackClients{URI: "https://stack.example", HTTPClient: &http.Client{}}, nil
		},
	})
	command := &cobra.Command{}
	command.SetContext(WithNonInteractive(context.Background()))

	client, err := factory(command)
	if err != nil {
		t.Fatalf("factory error = %v", err)
	}
	if client == nil {
		t.Fatal("client = nil, want Connectivity client")
	}
	if terminal.Len() != 0 {
		t.Fatalf("non-interactive dialog output = %q, want silence", terminal.String())
	}
}

func TestClientFactoryNormalModeRetainsInteractiveDialog(t *testing.T) {
	var terminal bytes.Buffer
	originalLogger := pterm.DefaultLogger
	pterm.DefaultLogger.Writer = &terminal
	t.Cleanup(func() { pterm.DefaultLogger = originalLogger })

	factory := newClientFactory(clientFactoryDependencies{
		loadAndAuthenticateCurrentProfile: testProfileLoader(),
		newStackClientsFromFlags: func(_ *cobra.Command, _ oidcclient.RelyingParty, dialog fctl.Dialog, _ string, _ fctl.Profile) (*fctl.StackClients, error) {
			dialog.Info("normal authentication path")
			return &fctl.StackClients{URI: "https://stack.example", HTTPClient: &http.Client{}}, nil
		},
	})

	client, err := factory(&cobra.Command{})
	if err != nil {
		t.Fatalf("factory error = %v", err)
	}
	if client == nil {
		t.Fatal("client = nil, want Connectivity client")
	}
	if !strings.Contains(terminal.String(), "normal authentication path") {
		t.Fatalf("normal dialog output = %q, want interactive message", terminal.String())
	}
}

func testProfileLoader() func(*cobra.Command) (*fctl.Config, *fctl.Profile, string, oidcclient.RelyingParty, error) {
	return func(*cobra.Command) (*fctl.Config, *fctl.Profile, string, oidcclient.RelyingParty, error) {
		return &fctl.Config{}, &fctl.Profile{}, "profile", nil, nil
	}
}

func accessTokenExpiringAt(expiration time.Time) *fctl.AccessToken {
	return &fctl.AccessToken{TokenWithClaims: fctl.TokenWithClaims[fctl.AccessTokenClaims]{
		Claims: fctl.AccessTokenClaims{TokenClaims: oidc.TokenClaims{Expiration: oidc.Time(expiration.Unix())}},
	}}
}

func TestSDKPatchPreservesExplicitMergePatchBody(t *testing.T) {
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/proxy/api/connectivity/connections/stripe", r.URL.Path)
		require.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))
		var err error
		received, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"stripe","configuration":{}}`)
	}))
	defer server.Close()
	body := []byte(`{"configuration":{"channel":null,"env":{"KEY":null},"files":{}}}`)
	ctx := WithPatchBody(t.Context(), body)
	response, err := NewAPI(server.URL+"/proxy", server.Client()).PatchConnection(ctx, operations.PatchConnectionRequest{Name: "stripe"}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.JSONEq(t, string(body), string(received))
}

func TestSDKResponseError(t *testing.T) {
	require.NoError(t, ResponseError(http.StatusNoContent, nil))
	err := ResponseError(http.StatusUnauthorized, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.Equal(t, "Unauthorized", apiErr.Message)
	err = ResponseError(http.StatusBadRequest, &connectivity.Error{Code: "invalid", Message: "bad", Details: map[string]any{"field": "name"}})
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, "invalid", apiErr.Code)
	require.Equal(t, map[string]any{"field": "name"}, apiErr.Details)
}

func TestSDKListConnectionsTransmitsQueryAndCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/connectivity/connections", r.URL.Path)
		require.Equal(t, "next page", r.URL.Query().Get("cursor"))
		require.Equal(t, "2", r.URL.Query().Get("pageSize"))
		require.Equal(t, `{"$match":{"connector":"stripe"}}`, r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"cursor":{"pageSize":2,"hasMore":true,"next":"next-token","data":[{"name":"stripe","configuration":{}}]}}`)
	}))
	defer server.Close()
	size := 2
	resp, err := NewAPI(server.URL, server.Client()).ListConnections(t.Context(), operations.ListConnectionsRequest{
		Cursor: OptionalString("next page"), PageSize: &size, Query: OptionalString(`{"$match":{"connector":"stripe"}}`),
	}, nil)
	require.NoError(t, err)
	require.NoError(t, ResponseError(resp.StatusCode, resp.Error))
	require.NotNil(t, resp.ConnectionCursor)
	require.True(t, resp.ConnectionCursor.Cursor.HasMore)
	require.Equal(t, "next-token", *resp.ConnectionCursor.Cursor.Next)
	require.Equal(t, "stripe", resp.ConnectionCursor.Cursor.Data[0].Name)
}

func TestSDKStructuredErrorUsesResponseStatusAndCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"invalid_query","message":"bad query","details":{"field":"query"}}`)
	}))
	defer server.Close()
	resp, err := NewAPI(server.URL, server.Client()).ListConnections(t.Context(), operations.ListConnectionsRequest{}, nil)
	require.NoError(t, err)
	var apiErr *APIError
	require.ErrorAs(t, ResponseError(resp.StatusCode, resp.Error), &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Equal(t, "invalid_query", apiErr.Code)
	require.Equal(t, map[string]any{"field": "query"}, apiErr.Details)
}
