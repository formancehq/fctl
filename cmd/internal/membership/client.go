package membership

import (
	"context"
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/debugutils"
	"github.com/formancehq/fctl/membershipclient"
)

func NewClient(ctx context.Context, cfg *config.Config) (*membershipclient.APIClient, error) {
	profile := config.GetCurrentProfile(ctx, cfg)
	httpClient := debugutils.GetHttpClient(ctx)
	configuration := membershipclient.NewConfiguration()
	token, err := profile.GetToken(ctx, httpClient)
	if err != nil {
		return nil, err
	}
	configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	configuration.HTTPClient = httpClient
	configuration.Servers[0].URL = profile.GetMembershipURI()
	return membershipclient.NewAPIClient(configuration), nil
}
