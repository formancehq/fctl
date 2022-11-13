package internal

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/numary/payments/client"
	"github.com/spf13/cobra"
)

func NewPaymentsClient(cmd *cobra.Command, cfg *config.Config) (*client.APIClient, error) {
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

	apiConfig := client.NewConfiguration()
	apiConfig.Servers = client.ServerConfigurations{{
		URL: profile.ApiUrl(stack, "payments").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return client.NewAPIClient(apiConfig), nil
}
