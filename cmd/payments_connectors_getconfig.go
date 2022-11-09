package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newPaymentsConnectorsGetConfigCommand() *cobra.Command {
	return newCommand("get-config [CONNECTOR_NAME]",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {

			config, err := getConfig()
			if err != nil {
				return err
			}

			client, err := newPaymentsClient(cmd, config)
			if err != nil {
				return err
			}

			connectorConfig, _, err := client.DefaultApi.ReadConnectorConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return errors.Wrap(err, "reding connector config")
			}
			switch args[0] {
			case "stripe":
				config := connectorConfig.StripeConfig
				fmt.Fprintln(cmd.OutOrStdout(), "Api key:", config.ApiKey)
				fmt.Fprintln(cmd.OutOrStdout(), "Polling period:", config.PollingPeriod)
				fmt.Fprintln(cmd.OutOrStdout(), "Page size:", config.PageSize)
			default:
				fmt.Fprintln(cmd.OutOrStdout(), "No specific output defined for connector", args[0])
				fmt.Fprintln(cmd.OutOrStdout(), config)
			}
			return nil
		}),
	)
}
