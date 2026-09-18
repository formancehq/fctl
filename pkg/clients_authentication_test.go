package fctl

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v4/oidc"
)

type authenticationDialog struct{}

func (authenticationDialog) Info(string, ...any) {}

func authenticationProfile(t *testing.T, expired bool) (*cobra.Command, Profile) {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String(ConfigDir, t.TempDir(), "")
	expires := time.Now().Add(time.Hour)
	if expired {
		expires = time.Now().Add(-time.Hour)
	}
	profile := Profile{
		MembershipURI: "https://oidc.example.test",
		RootTokens: &Tokens{
			Access: authenticationToken(t, "original-subject", expires),
			ID:     IDToken{Token: "synthetic-id-token"},
		},
		DefaultOrganization: "synthetic-organization",
		DefaultStack:        "synthetic-stack",
	}
	return cmd, profile
}

func TestEnsureMembershipAccessReusesValidToken(t *testing.T) {
	cmd, profile := authenticationProfile(t, false)
	// A nil RP makes any accidental refresh fail; no profile is written yet.
	got, err := EnsureMembershipAccess(cmd, nil, authenticationDialog{}, "test", profile)

	require.NoError(t, err)
	require.Equal(t, &profile.RootTokens.Access, got)
	_, err = os.Stat(GetFilePath(cmd, "profiles/test/profile.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestEnsureMembershipAccessRequiresConnectedProfile(t *testing.T) {
	cmd, profile := authenticationProfile(t, true)
	profile.RootTokens = nil

	got, err := EnsureMembershipAccess(cmd, nil, authenticationDialog{}, "test", profile)

	require.Nil(t, got)
	require.EqualError(t, err, "profile test is not connected, please log in")
}

func TestEnsureMembershipAccessPersistsRefreshedToken(t *testing.T) {
	for _, rotated := range []bool{true, false} {
		name := "refresh token omitted"
		if rotated {
			name = "refresh token rotated"
		}
		t.Run(name, func(t *testing.T) {
			cmd, profile := authenticationProfile(t, true)
			require.NoError(t, WriteProfile(cmd, "test", profile))
			want := authenticationToken(t, "refreshed-subject", time.Now().Add(time.Hour))
			response := map[string]string{"access_token": want.Token}
			if rotated {
				want.Refresh = "rotated-refresh-token"
				response["refresh_token"] = want.Refresh
			}
			calls := 0
			rp := authenticationRelyingParty(t, func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "synthetic-refresh-token", r.Form.Get("refresh_token"))
				return authenticationResponse(t, http.StatusOK, response), nil
			})

			got, err := EnsureMembershipAccess(cmd, rp, authenticationDialog{}, "test", profile)

			require.NoError(t, err)
			require.Equal(t, &want, got)
			require.Equal(t, 1, calls)
			persisted, err := LoadProfile(cmd, "test")
			require.NoError(t, err)
			require.Equal(t, want, persisted.RootTokens.Access)
			require.Equal(t, "synthetic-id-token", persisted.RootTokens.ID.Token)
			require.Equal(t, profile, *persisted)
		})
	}
}

func TestEnsureMembershipAccessReauthenticatesRejectedRefreshToken(t *testing.T) {
	for _, errorType := range []string{"invalid_token", "invalid_request"} {
		for _, authenticateFails := range []bool{false, true} {
			name := errorType + "/success"
			if authenticateFails {
				name = errorType + "/failure"
			}
			t.Run(name, func(t *testing.T) {
				cmd, profile := authenticationProfile(t, true)
				require.NoError(t, WriteProfile(cmd, "test", profile))
				original, err := os.ReadFile(GetFilePath(cmd, "profiles/test/profile.json"))
				require.NoError(t, err)
				refreshCalls := 0
				rp := authenticationRelyingParty(t, func(*http.Request) (*http.Response, error) {
					refreshCalls++
					return authenticationResponse(t, http.StatusBadRequest, map[string]string{"error": errorType}), nil
				})
				want := authenticationToken(t, "reauthenticated-subject", time.Now().Add(time.Hour))
				want.Refresh = "reauthenticated-refresh-token"
				authenticateErr := errors.New("synthetic authentication failure")
				authenticateCalls := 0
				authenticate := func() (*Tokens, error) {
					authenticateCalls++
					if authenticateFails {
						return nil, authenticateErr
					}
					return &Tokens{Access: want}, nil
				}

				got, err := ensureMembershipAccess(cmd, rp, authenticationDialog{}, "test", profile, authenticate)

				require.Equal(t, 1, refreshCalls)
				require.Equal(t, 1, authenticateCalls)
				if authenticateFails {
					require.Nil(t, got)
					require.ErrorIs(t, err, authenticateErr)
					require.ErrorContains(t, err, "failed to authenticate for membership")
					persisted, err := os.ReadFile(GetFilePath(cmd, "profiles/test/profile.json"))
					require.NoError(t, err)
					require.Equal(t, original, persisted)
					return
				}
				require.NoError(t, err)
				require.Equal(t, &want, got)
				persisted, err := LoadProfile(cmd, "test")
				require.NoError(t, err)
				require.Equal(t, want, persisted.RootTokens.Access)
				require.Equal(t, "synthetic-id-token", persisted.RootTokens.ID.Token)
				require.Equal(t, profile, *persisted)
			})
		}
	}
}

func TestEnsureMembershipAccessPreservesRefreshErrors(t *testing.T) {
	transportErr := errors.New("synthetic connection failure")
	for _, tc := range []struct {
		name      string
		oauthType string
		cause     error
	}{
		{name: "invalid client", oauthType: "invalid_client"},
		{name: "server error", oauthType: "server_error"},
		{name: "invalid grant", oauthType: "invalid_grant"},
		{name: "network failure", cause: transportErr},
		{name: "cancellation", cause: context.Canceled},
		{name: "malformed access token", cause: oidc.ErrParse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, profile := authenticationProfile(t, true)
			require.NoError(t, WriteProfile(cmd, "test", profile))
			original, err := os.ReadFile(GetFilePath(cmd, "profiles/test/profile.json"))
			require.NoError(t, err)
			if tc.cause == context.Canceled {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				cmd.SetContext(ctx)
			}
			rp := authenticationRelyingParty(t, func(r *http.Request) (*http.Response, error) {
				switch {
				case tc.oauthType != "":
					return authenticationResponse(t, http.StatusBadRequest, map[string]string{"error": tc.oauthType}), nil
				case tc.cause == oidc.ErrParse:
					return authenticationResponse(t, http.StatusOK, map[string]string{"access_token": "not-a-jwt"}), nil
				case tc.cause == context.Canceled:
					return nil, r.Context().Err()
				default:
					return nil, tc.cause
				}
			})

			got, err := EnsureMembershipAccess(cmd, rp, authenticationDialog{}, "test", profile)

			require.Nil(t, got)
			if tc.oauthType != "" {
				var oauthErr *oidc.Error
				require.ErrorAs(t, err, &oauthErr)
				require.Equal(t, tc.oauthType, string(oauthErr.ErrorType))
				require.ErrorContains(t, err, "received unexpected oauth2 error")
			} else {
				require.ErrorIs(t, err, tc.cause)
				require.ErrorContains(t, err, "failed to refresh membership token")
			}
			persisted, err := os.ReadFile(GetFilePath(cmd, "profiles/test/profile.json"))
			require.NoError(t, err)
			require.Equal(t, original, persisted)
		})
	}
}

func TestEnsureMembershipAccessReportsPersistenceError(t *testing.T) {
	cmd, profile := authenticationProfile(t, true)
	// A file in place of the profile directory makes persistence fail reliably.
	require.NoError(t, os.WriteFile(filepath.Join(LoadConfigDir(cmd), "profiles"), []byte("blocked"), 0600))
	want := authenticationToken(t, "refreshed-subject", time.Now().Add(time.Hour))
	rp := authenticationRelyingParty(t, func(*http.Request) (*http.Response, error) {
		return authenticationResponse(t, http.StatusOK, map[string]string{"access_token": want.Token}), nil
	})

	got, err := EnsureMembershipAccess(cmd, rp, authenticationDialog{}, "test", profile)

	require.Nil(t, got)
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
}
