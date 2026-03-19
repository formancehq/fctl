package configs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/connectors/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
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
