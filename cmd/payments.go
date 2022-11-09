package cmd

import (
	"fmt"

	fctl "github.com/formancehq/fctl/cmd/internal"
	"github.com/numary/payments/client"
	"github.com/spf13/cobra"
)

func newPaymentsCommand() *cobra.Command {
	return newStackCommand("payments",
		withChildCommands(
			newPaymentsConnectorsCommand(),
		),
	)
}

func newPaymentsClient(cmd *cobra.Command, config *fctl.Config) (*client.APIClient, error) {
	profile, err := getCurrentProfile(config)
	if err != nil {
		return nil, err
	}

	organizationID, err := resolveOrganizationID(cmd, config)
	if err != nil {
		return nil, err
	}

	stackID, err := resolveStackID(cmd, config, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := getHttpClient()

	token, err := profile.GetToken(cmd.Context(), httpClient)
	if err != nil {
		return nil, err
	}

	apiConfig := client.NewConfiguration()
	apiConfig.Servers = client.ServerConfigurations{{
		URL: profile.ApiUrl(organizationID, stackID, "payments").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return client.NewAPIClient(apiConfig), nil
}
