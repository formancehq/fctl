package cmd

import (
	"github.com/numary/payments/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newPaymentsConnectorsInstallStripeCommand() *cobra.Command {
	const (
		stripeApiKeyFlag = "api-key"
	)
	return newCommand("stripe [API_KEY]",
		withShortDescription("Install a stripe connector"),
		withArgs(cobra.ExactArgs(1)),
		withStringFlag(stripeApiKeyFlag, "", "Stripe API key"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			paymentsClient, err := newPaymentsClient(cmd)
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
