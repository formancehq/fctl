package install

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/payments/internal"
	"github.com/formancehq/payments/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewStripeCommand() *cobra.Command {
	const (
		stripeApiKeyFlag = "api-key"
	)
	return cmdbuilder.NewCommand("stripe [API_KEY]",
		cmdbuilder.WithShortDescription("Install a stripe connector"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithStringFlag(stripeApiKeyFlag, "", "Stripe API key"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
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

			cmdbuilder.Success(cmd.OutOrStdout(), "Connector installed!")

			return errors.Wrap(err, "installing connector")
		}),
	)
}
