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
			config, err := getConfig()
			if err != nil {
				return err
			}
			paymentsClient, err := newPaymentsClient(cmd, config)
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
