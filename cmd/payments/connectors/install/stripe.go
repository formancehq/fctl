package install

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewStripeCommand() *cobra.Command {
	const (
		stripeApiKeyFlag = "api-key"
	)
	return fctl.NewCommand("stripe [API_KEY]",
		fctl.WithShortDescription("Install a stripe connector"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag(stripeApiKeyFlag, "", "Stripe API key"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.Get(cmd)
			if err != nil {
				return err
			}

			paymentsClient, err := fctl.NewStackClient(cmd, cfg)
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

			fctl.Success(cmd.OutOrStdout(), "Connector installed!")

			return errors.Wrap(err, "installing connector")
		}),
	)
}
