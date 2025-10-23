package configs

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/payments/connectors/internal"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewUpdateConfigCommands() *cobra.Command {
	return fctl.NewCommand("update-config",
		fctl.WithAliases("uc"),
		fctl.WithShortDescription(fmt.Sprintf("Update the config of a connector (Connectors available: %v)", internal.AllConnectors)),
		fctl.WithChildCommands(
			newUpdateAdyenCommand(),
			newUpdateAtlarCommand(),
			newUpdateBankingCircleCommand(),
			newUpdateCurrencyCloudCommand(),
			newUpdateMangopayCommand(),
			newUpdateModulrCommand(),
			newUpdateMoneycorpCommand(),
			newUpdateStripeCommand(),
			newUpdateWiseCommand(),
			newUpdateQontoCommand(),
			newUpdateColumnCommand(),
		),
	)
}
