package payments

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/accounts"
	"github.com/formancehq/fctl/v3/cmd/payments/bankaccounts"
	"github.com/formancehq/fctl/v3/cmd/payments/connectors"
	"github.com/formancehq/fctl/v3/cmd/payments/payments"
	"github.com/formancehq/fctl/v3/cmd/payments/pools"
	"github.com/formancehq/fctl/v3/cmd/payments/tasks"
	"github.com/formancehq/fctl/v3/cmd/payments/transferinitiation"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("payments",
		fctl.WithShortDescription("Payments management"),
		fctl.WithChildCommands(
			connectors.NewConnectorsCommand(),
			payments.NewPaymentsCommand(),
			transferinitiation.NewTransferInitiationCommand(),
			bankaccounts.NewBankAccountsCommand(),
			accounts.NewAccountsCommand(),
			pools.NewPoolsCommand(),
			tasks.NewTasksCommand(),
		),
	)
}
