package cmd

import (
	"fmt"

	"github.com/formancehq/auth/authclient"
	fctl "github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	return newStackCommand("auth",
		withChildCommands(
			newAuthClientsCommand(),
		),
	)
}

func newAuthClient(cmd *cobra.Command, config *fctl.Config) (*authclient.APIClient, error) {
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
