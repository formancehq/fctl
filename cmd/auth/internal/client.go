package internal

import (
	"fmt"

	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewAuthClient(cmd *cobra.Command, cfg *config.Config) (*authclient.APIClient, error) {
	profile := config.GetCurrentProfile(cmd, cfg)

	organizationID, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
	if err != nil {
		return nil, err
	}

	stack, err := cmdbuilder.ResolveStack(cmd, cfg, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := config.GetHttpClient(cmd)

	token, err := profile.GetStackToken(cmd.Context(), httpClient, stack)
	if err != nil {
		return nil, err
	}

	apiConfig := authclient.NewConfiguration()
	apiConfig.Servers = authclient.ServerConfigurations{{
		URL: profile.ApiUrl(stack, "auth").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return authclient.NewAPIClient(apiConfig), nil
}
