package internal

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/debugutil"
	"github.com/numary/payments/client"
	"github.com/spf13/cobra"
)

func NewPaymentsClient(cmd *cobra.Command, cfg *config.Config) (*client.APIClient, error) {
	profile := config.GetCurrentProfile(cfg)

	organizationID, err := cmdbuilder.ResolveOrganizationID(cmd.Context(), cfg)
	if err != nil {
		return nil, err
	}

	stackID, err := cmdbuilder.ResolveStackID(cmd.Context(), cfg, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := debugutil.GetHttpClient()

	token, err := profile.GetToken(cmd.Context(), httpClient)
	if err != nil {
		return nil, err
	}

	apiConfig := client.NewConfiguration()
	apiConfig.Servers = client.ServerConfigurations{{
		URL: profile.ApiUrl(organizationID, stackID, "payments").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	apiConfig.HTTPClient = httpClient

	return client.NewAPIClient(apiConfig), nil
}
