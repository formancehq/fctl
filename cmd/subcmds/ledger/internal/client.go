package internal

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/internal/debugutil"
	"github.com/numary/ledger/client"
	"github.com/spf13/cobra"
)

func NewLedgerClient(cmd *cobra.Command, cfg *config.Config) (*client.APIClient, error) {
	profile, err := config.GetCurrentProfile(cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
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

	apiConfig := client.NewConfiguration()
	apiConfig.Servers = client.ServerConfigurations{{
		URL: profile.ApiUrl(organizationID, stackID, "ledger").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return client.NewAPIClient(apiConfig), nil
}
