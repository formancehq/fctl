package cmd

import (
	"context"

	"github.com/formancehq/fctl/pkg/payments"
	"github.com/formancehq/fctl/pkg/stack"
	"github.com/numary/payments/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getPaymentsClient(ctx context.Context) (*client.APIClient, error) {
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

	return payments.NewClient(currentProfile, viper.GetBool(debugFlag), organization, stackId, token), nil
}

var paymentsCommand = &cobra.Command{
	Use: "payments",
}

func init() {
	paymentsCommand.PersistentFlags().String(stackFlag, "", "Specific stack (not required if only one stack is present)")
	paymentsCommand.PersistentFlags().String(ledgerFlag, "default", "Specific ledger ")

	rootCommand.AddCommand(paymentsCommand)
}
