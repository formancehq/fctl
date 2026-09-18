package fctl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v4/oidc"
	"github.com/formancehq/go-libs/v4/oidc/client"
)

type authenticationTransport func(*http.Request) (*http.Response, error)

func (f authenticationTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func authenticationResponse(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(data))),
	}
}

// All OIDC requests stay in this transport, including discovery. Unexpected
// device authorization requests fail before Authenticate could open a browser.
func authenticationRelyingParty(t *testing.T, tokenEndpoint authenticationTransport) client.RelyingParty {
	t.Helper()
	const issuer = "https://oidc.example.test"
	httpClient := &http.Client{Transport: authenticationTransport(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case issuer + "/.well-known/openid-configuration":
			return authenticationResponse(t, http.StatusOK, map[string]string{
				"issuer":                        issuer,
				"token_endpoint":                issuer + "/token",
				"device_authorization_endpoint": issuer + "/device",
			}), nil
		case issuer + "/token":
			require.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, r.ParseForm())
			require.Equal(t, "refresh_token", r.Form.Get("grant_type"))
			require.Equal(t, AuthClient, r.Form.Get("client_id"))
			return tokenEndpoint(r)
		default:
			t.Errorf("unexpected OIDC request: %s", r.URL)
			return nil, errors.New("unexpected OIDC request")
		}
	})}
	rp, err := GetAuthRelyingParty(context.Background(), httpClient, issuer)
	require.NoError(t, err)
	return rp
}

func authenticationToken(t *testing.T, subject string, expires time.Time) AccessToken {
	t.Helper()
	claims := AccessTokenClaims{
		TokenClaims: oidc.TokenClaims{
			Subject:    subject,
			Expiration: oidc.Time(expires.Unix()),
		},
		Scopes:         oidc.SpaceDelimitedArray{"accesses", "on_behalf"},
		OrganizationID: "synthetic-organization",
	}
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	// Refresh only decodes access-token claims; signature verification belongs to
	// the OIDC library. No ID token is included in these refresh responses.
	jwt := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "."
	return AccessToken{
		TokenWithClaims: TokenWithClaims[AccessTokenClaims]{Token: jwt, Claims: claims},
		Refresh:         "synthetic-refresh-token",
	}
}

func TestRefreshUpdatesAccessToken(t *testing.T) {
	for _, rotated := range []bool{true, false} {
		name := "preserves refresh token when omitted"
		if rotated {
			name = "replaces rotated refresh token"
		}
		t.Run(name, func(t *testing.T) {
			original := authenticationToken(t, "old-subject", time.Now().Add(-time.Hour))
			want := authenticationToken(t, "new-subject", time.Now().Add(time.Hour))
			response := map[string]any{"access_token": want.Token, "token_type": "Bearer", "expires_in": 3600}
			if rotated {
				want.Refresh = "rotated-refresh-token"
				response["refresh_token"] = want.Refresh
			}
			calls := 0
			rp := authenticationRelyingParty(t, func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, original.Refresh, r.Form.Get("refresh_token"))
				return authenticationResponse(t, http.StatusOK, response), nil
			})

			got, err := Refresh(context.Background(), rp, original)

			require.NoError(t, err)
			require.Equal(t, &want, got)
			require.Equal(t, 1, calls)
			require.Equal(t, "old-subject", original.Claims.Subject)
			require.Equal(t, "synthetic-refresh-token", original.Refresh)
		})
	}
}

func TestRefreshPreservesOAuthErrors(t *testing.T) {
	for _, errorType := range []string{"invalid_token", "invalid_request", "invalid_client", "server_error"} {
		t.Run(errorType, func(t *testing.T) {
			rp := authenticationRelyingParty(t, func(*http.Request) (*http.Response, error) {
				return authenticationResponse(t, http.StatusBadRequest, map[string]string{
					"error": errorType, "error_description": "synthetic OAuth failure",
				}), nil
			})

			got, err := Refresh(context.Background(), rp, AccessToken{Refresh: "synthetic-refresh-token"})

			require.Nil(t, got)
			require.True(t, IsInvalidAuthentication(err))
			var oauthErr *oidc.Error
			require.ErrorAs(t, err, &oauthErr)
			require.Equal(t, errorType, string(oauthErr.ErrorType))
			require.Equal(t, "synthetic OAuth failure", oauthErr.Description)
		})
	}
}

func TestRefreshRejectsMalformedAccessToken(t *testing.T) {
	rp := authenticationRelyingParty(t, func(*http.Request) (*http.Response, error) {
		return authenticationResponse(t, http.StatusOK, map[string]string{
			"access_token": "not-a-jwt", "refresh_token": "rotated-refresh-token",
		}), nil
	})

	got, err := Refresh(context.Background(), rp, AccessToken{Refresh: "synthetic-refresh-token"})

	require.Nil(t, got)
	require.ErrorIs(t, err, oidc.ErrParse)
	require.True(t, IsInvalidAuthentication(err))
	var oauthErr *oidc.Error
	require.False(t, errors.As(err, &oauthErr))
}

func TestRefreshPreservesTransportError(t *testing.T) {
	transportErr := errors.New("synthetic connection failure")
	rp := authenticationRelyingParty(t, func(*http.Request) (*http.Response, error) {
		return nil, transportErr
	})

	got, err := Refresh(context.Background(), rp, AccessToken{Refresh: "synthetic-refresh-token"})

	require.Nil(t, got)
	require.ErrorIs(t, err, transportErr)
	require.True(t, IsInvalidAuthentication(err))
}

func TestRefreshCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rp := authenticationRelyingParty(t, func(r *http.Request) (*http.Response, error) {
		cancel()
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case <-time.After(5 * time.Second):
			return nil, errors.New("refresh request did not observe cancellation")
		}
	})

	got, err := Refresh(ctx, rp, AccessToken{Refresh: "synthetic-refresh-token"})

	require.Nil(t, got)
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, IsInvalidAuthentication(err))
}
