package fctl

import (
	"context"
	"fmt"
	"net/http"

	"github.com/numary/membership-api/client"
)

func NewMembershipClientFromContext(ctx context.Context, profile *Profile, httpClient *http.Client) (*client.APIClient, error) {
	configuration := client.NewConfiguration()
	token, err := profile.GetToken(ctx, httpClient)
	if err != nil {
		return nil, err
	}
	configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	configuration.HTTPClient = httpClient
	configuration.Servers[0].URL = profile.MembershipURI
	return client.NewAPIClient(configuration), nil
}
