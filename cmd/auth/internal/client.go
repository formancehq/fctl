package internal

import (
	"fmt"

	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/debugutil"
	"github.com/spf13/cobra"
)

func NewAuthClient(cmd *cobra.Command, cfg *config.Config) (*authclient.APIClient, error) {
	profile, err := config.GetCurrentProfile(cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := cmdbuilder.ResolveOrganizationID(cmd.Context(), cfg)
	if err != nil {
		return nil, err
	}

	stackID, err := cmdbuilder.ResolveStackID(cmd, cfg, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := debugutil.GetHttpClient()

	token, err := profile.GetStackToken(cmd.Context(), httpClient, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	apiConfig := authclient.NewConfiguration()
	apiConfig.Servers = authclient.ServerConfigurations{{
		URL: profile.ApiUrl(organizationID, stackID, "auth").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return authclient.NewAPIClient(apiConfig), nil
}
