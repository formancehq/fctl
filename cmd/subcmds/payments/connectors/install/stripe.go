package install

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/payments/internal"
	"github.com/numary/payments/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewPaymentsConnectorsInstallStripeCommand() *cobra.Command {
	const (
		stripeApiKeyFlag = "api-key"
	)
	return cmdbuilder.NewCommand("stripe [API_KEY]",
		cmdbuilder.WithShortDescription("Install a stripe connector"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithStringFlag(stripeApiKeyFlag, "", "Stripe API key"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			paymentsClient, err := internal.NewPaymentsClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = paymentsClient.DefaultApi.InstallConnector(cmd.Context(), "stripe").
				ConnectorConfig(client.ConnectorConfig{
					StripeConfig: &client.StripeConfig{
						ApiKey: args[0],
					},
				}).
				Execute()

			return errors.Wrap(err, "installing connector")
		}),
	)
}
