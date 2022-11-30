package install

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewStripeCommand() *cobra.Command {
	const (
		stripeApiKeyFlag = "api-key"
	)
	return internal2.NewCommand("stripe [API_KEY]",
		internal2.WithShortDescription("Install a stripe connector"),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithStringFlag(stripeApiKeyFlag, "", "Stripe API key"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			paymentsClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = paymentsClient.PaymentsApi.InstallConnector(cmd.Context(), "stripe").
				ConnectorConfig(formance.ConnectorConfig{
					StripeConfig: &formance.StripeConfig{
						ApiKey: args[0],
					},
				}).
				Execute()

			internal2.Success(cmd.OutOrStdout(), "Connector installed!")

			return errors.Wrap(err, "installing connector")
		}),
	)
}
