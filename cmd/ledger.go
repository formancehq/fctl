package cmd

import (
	"context"

	"github.com/formancehq/fctl/pkg/ledger"
	"github.com/formancehq/fctl/pkg/stack"
	ledgerclient "github.com/numary/ledger/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ledgerFlag = "ledger"
)

func getLedgerClient(ctx context.Context) (*ledgerclient.APIClient, error) {
	organization, err := findOrganizationId(ctx)
	if err != nil {
		return nil, err
	}

	stackId, err := findStackId(ctx, organization)
	if err != nil {
		return nil, err
	}

	token, err := stack.GetToken(ctx, *currentProfile, organization, stackId)
	if err != nil {
		return nil, err
	}

	return ledger.NewClient(*currentProfile, viper.GetBool(debugFlag), organization, stackId, token), nil
}

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
