package install

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewInstallCommand() *cobra.Command {
	return fctl.NewCommand("install",
		fctl.WithAliases("i"),
		fctl.WithShortDescription(fmt.Sprintf("Install a connector (Connectors available: %v)", internal.AllConnectors)),
		fctl.WithChildCommands(
			NewAdyenCommand(),
			NewStripeCommand(),
			NewBankingCircleCommand(),
			NewCurrencyCloudCommand(),
			NewModulrCommand(),
			NewWiseCommand(),
			NewMangoPayCommand(),
			NewMoneycorpCommand(),
			NewAtlarCommand(),
			NewGenericCommand(),
			NewQontoCommand(),
			NewColumnCommand(),
		),
	)
}
