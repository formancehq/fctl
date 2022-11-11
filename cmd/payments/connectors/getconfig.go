package connectors

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/payments/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewPaymentsConnectorsGetConfigCommand() *cobra.Command {
	return cmdbuilder.NewCommand("get-config [CONNECTOR_NAME]",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithShortDescription("Read a connector config"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get()
			if err != nil {
				return err
			}

			client, err := internal.NewPaymentsClient(cmd, cfg)
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
				fmt.Fprintln(cmd.OutOrStdout(), cfg)
			}
			return nil
		}),
	)
}
