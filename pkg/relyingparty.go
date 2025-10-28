package fctl

import (
	"context"
	"net/http"

	"github.com/formancehq/go-libs/v3/oidc/client"
	"github.com/pterm/pterm"
)

func GetAuthRelyingParty(ctx context.Context, httpClient *http.Client, membershipURI string) (client.RelyingParty, error) {
	pterm.Debug.Println("Getting auth relying party on membership URI:", membershipURI)
	return client.NewRelyingPartyOIDC(
		ctx,
		membershipURI,
		AuthClient,
		"",
		"",
		[]string{},
		client.WithHTTPClient(httpClient),
	)
}
