package cmd

import (
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

func newLedgerClient(cmd *cobra.Command) (*ledgerclient.APIClient, error) {
	profile, err := getCurrentProfile()
	if err != nil {
		return nil, err
	}
	organizationID, err := resolveOrganizationID(cmd)
	if err != nil {
		return nil, err
	}
	stackID, err := resolveStackID(cmd, organizationID)
	if err != nil {
		return nil, err
	}
	return fctl.NewLedgerClientFromContext(cmd.Context(), profile, getHttpClient(),
		organizationID, stackID)
}
