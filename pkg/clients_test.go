package fctl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/formancehq/go-libs/v4/oidc"
	"github.com/formancehq/go-libs/v4/oidc/client"
)

type tokenTestRelyingParty struct {
	client.RelyingParty
	httpClient  *http.Client
	oauthConfig *oauth2.Config
}

func (r tokenTestRelyingParty) HttpClient() *http.Client {
	return r.httpClient
}

func (r tokenTestRelyingParty) OAuthConfig() *oauth2.Config {
	return r.oauthConfig
}

func TestStackTokenSourceDeadlineBoundsStackAPITokenFetch(t *testing.T) {
	tokenStarted := make(chan struct{})
	releaseToken := make(chan struct{})
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/api/auth/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"token_endpoint":%q}`, server.URL+"/token")
		case "/token":
			close(tokenStarted)
			select {
			case <-req.Context().Done():
			case <-releaseToken:
			}
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	cmd := tokenSourceTestCommand(t, ctx)
	stackToken := accessTokenExpiringAt(time.Now().Add(time.Hour))
	source := NewStackTokenSource(
		stackToken,
		&StackAccess{URI: server.URL},
		tokenTestRelyingParty{httpClient: server.Client()},
		func(AccessToken) error { return nil },
		cmd,
		"profile",
		"organization",
		"stack",
	)

	result := make(chan error, 1)
	go func() {
		_, err := source.Token()
		result <- err
	}()
	select {
	case <-tokenStarted:
	case <-time.After(time.Second):
		close(releaseToken)
		t.Fatal("stack API token endpoint was not called")
	}

	select {
	case err := <-result:
		require.ErrorIs(t, err, context.DeadlineExceeded)
		close(releaseToken)
	case <-time.After(300 * time.Millisecond):
		close(releaseToken)
		<-result
		t.Fatal("stack API token acquisition outlived the command deadline")
	}
}

func TestStackTokenSourceCancellationBoundsStackTokenRefresh(t *testing.T) {
	tokenStarted := make(chan struct{})
	releaseToken := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/token" {
			http.NotFound(w, req)
			return
		}
		close(tokenStarted)
		select {
		case <-req.Context().Done():
		case <-releaseToken:
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := tokenSourceTestCommand(t, ctx)
	source := NewStackTokenSource(
		accessTokenExpiringAt(time.Now().Add(-time.Hour)),
		&StackAccess{URI: server.URL},
		tokenTestRelyingParty{
			httpClient: server.Client(),
			oauthConfig: &oauth2.Config{
				ClientID: "fctl",
				Endpoint: oauth2.Endpoint{TokenURL: server.URL + "/token"},
			},
		},
		func(AccessToken) error { return nil },
		cmd,
		"profile",
		"organization",
		"stack",
	)

	result := make(chan error, 1)
	go func() {
		_, err := source.Token()
		result <- err
	}()
	select {
	case <-tokenStarted:
	case <-time.After(time.Second):
		close(releaseToken)
		t.Fatal("stack refresh token endpoint was not called")
	}
	cancel()

	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
		close(releaseToken)
	case <-time.After(300 * time.Millisecond):
		close(releaseToken)
		<-result
		t.Fatal("stack token refresh outlived command cancellation")
	}
}

func tokenSourceTestCommand(t *testing.T, ctx context.Context) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.Flags().String(ConfigDir, t.TempDir(), "")
	return cmd
}

func accessTokenExpiringAt(expiry time.Time) AccessToken {
	return AccessToken{
		TokenWithClaims: TokenWithClaims[AccessTokenClaims]{
			Token: "stack-token",
			Claims: AccessTokenClaims{
				TokenClaims: oidc.TokenClaims{Expiration: oidc.Time(expiry.Unix())},
			},
		},
		Refresh: "refresh-token",
	}
}
