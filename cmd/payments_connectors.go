package cmd

import (
	"fmt"

	"github.com/numary/payments/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var connectorsCommand = &cobra.Command{
	Use:   "connectors",
	Short: "Handle payments service connectors",
}

var connectorsGetConfigCommand = &cobra.Command{
	Use:  "get-config [CONNECTOR_NAME]",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getPaymentsClient(cmd.Context())
		if err != nil {
			return err
		}

		config, _, err := client.DefaultApi.ReadConnectorConfig(cmd.Context(), args[0]).Execute()
		if err != nil {
			return errors.Wrap(err, "reding connector config")
		}
		switch args[0] {
		case "stripe":
			config := config.StripeConfig
			fmt.Fprintln(cmd.OutOrStdout(), "Api key:", config.ApiKey)
			fmt.Fprintln(cmd.OutOrStdout(), "Polling period:", config.PollingPeriod)
			fmt.Fprintln(cmd.OutOrStdout(), "Page size:", config.PageSize)
		default:
			fmt.Fprintln(cmd.OutOrStdout(), "No specific output defined for connector", args[0])
			fmt.Fprintln(cmd.OutOrStdout(), config)
		}
		return nil
	},
}

var installConnectorCommand = &cobra.Command{
	Use:   "install",
	Short: "Install a connector",
}

const (
	stripeApiKeyFlag = "api-key"
)

var installStripeConnectorCommand = &cobra.Command{
	Use:   "stripe [API_KEY]",
	Short: "Install a stripe connector",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		paymentsClient, err := getPaymentsClient(cmd.Context())
		if err != nil {
			return err
		}
		_, err = paymentsClient.DefaultApi.InstallConnector(cmd.Context(), "stripe").ConnectorConfig(client.ConnectorConfig{
			StripeConfig: &client.StripeConfig{
				ApiKey: args[0],
			},
		}).Execute()

		return errors.Wrap(err, "installing connector")
	},
}

func init() {
	installStripeConnectorCommand.Flags().String(stripeApiKeyFlag, "", "Stripe API key")
	installConnectorCommand.AddCommand(installStripeConnectorCommand)
	connectorsCommand.AddCommand(connectorsGetConfigCommand, installConnectorCommand)
	paymentsCommand.AddCommand(connectorsCommand)
}
