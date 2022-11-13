package connectors

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/openapi"
	"github.com/formancehq/fctl/cmd/payments/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewPaymentsConnectorsGetConfigCommand() *cobra.Command {
	return cmdbuilder.NewCommand("get-config [CONNECTOR_NAME]",
		cmdbuilder.WithAliases("getconfig", "getconf", "gc", "get", "g"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithValidArgs("stripe"),
		cmdbuilder.WithShortDescription("Read a connector config"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal.NewPaymentsClient(cmd, cfg)
			if err != nil {
				return err
			}

			connectorConfig, _, err := client.DefaultApi.ReadConnectorConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return openapi.WrapError(err, "reading connector config")
			}
			switch args[0] {
			case "stripe":
				config := connectorConfig.StripeConfig

				tableData := pterm.TableData{}
				tableData = append(tableData, []string{pterm.LightCyan("Api key:"), config.ApiKey})

				if err := pterm.DefaultTable.
					WithWriter(cmd.OutOrStdout()).
					WithData(tableData).
					Render(); err != nil {
					return err
				}
			default:
				cmdbuilder.Error(cmd.ErrOrStderr(), "Connection unknown.")
			}
			return nil
		}),
	)
}
