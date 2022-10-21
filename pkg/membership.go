package fctl

import (
	"context"
	"fmt"

	"github.com/numary/membership-api/client"
)

func NewMembershipClientFromContext(ctx context.Context) *client.APIClient {
	profile := CurrentProfileFromContext(ctx)
	configuration := client.NewConfiguration()
	if profile.Token != nil {
		configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", profile.Token.AccessToken))
	}
	configuration.HTTPClient = HttpClientFromContext(ctx)
	configuration.Servers[0].URL = profile.MembershipURI
	return client.NewAPIClient(configuration)
}
