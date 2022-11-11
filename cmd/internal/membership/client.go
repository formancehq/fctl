package membership

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/internal/debugutil"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

func NewMembershipClient(cmd *cobra.Command, cfg *config.Config) (*membershipclient.APIClient, error) {
	profile, err := config.GetCurrentProfile(cfg)
	if err != nil {
		return nil, err
	}

	httpClient := debugutil.GetHttpClient()
	configuration := membershipclient.NewConfiguration()
	token, err := profile.GetToken(cmd.Context(), httpClient)
	if err != nil {
		return nil, err
	}
	configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	configuration.HTTPClient = httpClient
	configuration.Servers[0].URL = profile.GetMembershipURI()
	return membershipclient.NewAPIClient(configuration), nil
}
