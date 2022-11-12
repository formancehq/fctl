package config

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

func NewClient(cmd *cobra.Command, cfg *Config) (*membershipclient.APIClient, error) {
	profile := GetCurrentProfile(cmd, cfg)
	httpClient := GetHttpClient(cmd)
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
