package cmd

import (
	"fmt"

	fctl "github.com/formancehq/fctl/cmd/internal"
	ledgerclient "github.com/numary/ledger/client"
	"github.com/spf13/cobra"
)

const (
	ledgerFlag = "ledger"
)

func newLedgerCommand() *cobra.Command {
	return newStackCommand("ledger",
		withPersistentStringFlag(ledgerFlag, "default", "Specific ledger"),
		withChildCommands(
			newLedgerTransactionsCommand(),
			newLedgerBalancesCommand(),
			newLedgerAccountsCommand(),
		),
	)
}

func newLedgerClient(cmd *cobra.Command, config *fctl.Config) (*ledgerclient.APIClient, error) {
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

	apiConfig := ledgerclient.NewConfiguration()
	apiConfig.Servers = ledgerclient.ServerConfigurations{{
		URL: profile.ApiUrl(organizationID, stackID, "ledger").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return ledgerclient.NewAPIClient(apiConfig), nil
}
