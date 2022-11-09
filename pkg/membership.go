package fctl

import (
	"context"
	"fmt"

	"github.com/numary/membership-api/client"
)

func NewMembershipClientFromContext(ctx context.Context) (*client.APIClient, error) {
	profile := CurrentProfileFromContext(ctx)
	configuration := client.NewConfiguration()
	token, err := profile.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	configuration.HTTPClient = NewHTTPClientFromContext(ctx)
	configuration.Servers[0].URL = profile.MembershipURI
	return client.NewAPIClient(configuration), nil
}
