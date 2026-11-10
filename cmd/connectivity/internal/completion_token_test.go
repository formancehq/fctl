package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v4/oidc/client"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func completionTokenCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("config-dir", t.TempDir(), "")
	cmd.SetContext(t.Context())
	return cmd
}

func stubRefresh(t *testing.T, refresh func(context.Context, client.RelyingParty, fctl.AccessToken) (*fctl.AccessToken, error)) {
	t.Helper()
	previous := refreshToken
	refreshToken = refresh
	t.Cleanup(func() { refreshToken = previous })
}

func TestRefreshStackTokenGivenAValidRefreshTokenThenStoresTheRenewedToken(t *testing.T) {
	cmd := completionTokenCommand(t)
	expired := fctl.AccessToken{Refresh: "refresh-1"}
	expired.Claims.OrganizationID = "org"
	renewed := fctl.AccessToken{Refresh: "refresh-2"}
	renewed.Token = "renewed"
	renewed.Claims.OrganizationID = "org"
	stubRefresh(t, func(_ context.Context, _ client.RelyingParty, token fctl.AccessToken) (*fctl.AccessToken, error) {
		require.Equal(t, "refresh-1", token.Refresh)
		return &renewed, nil
	})

	ok := refreshStackToken(cmd, nil, "default", "stack", expired)

	require.True(t, ok)
	stored, err := fctl.ReadStackToken(cmd, "default", "org", "stack")
	require.NoError(t, err)
	require.Equal(t, "renewed", stored.Token)
}

func TestRefreshStackTokenGivenARejectedRefreshThenOffersNothingAndKeepsTheStore(t *testing.T) {
	cmd := completionTokenCommand(t)
	expired := fctl.AccessToken{Refresh: "refresh-1"}
	expired.Claims.OrganizationID = "org"
	stubRefresh(t, func(context.Context, client.RelyingParty, fctl.AccessToken) (*fctl.AccessToken, error) {
		return nil, errors.New("invalid_grant")
	})

	ok := refreshStackToken(cmd, nil, "default", "stack", expired)

	require.False(t, ok)
	stored, err := fctl.ReadStackToken(cmd, "default", "org", "stack")
	require.NoError(t, err)
	require.Nil(t, stored)
}
