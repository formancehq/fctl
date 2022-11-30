package connectors

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewGetConfigCommand() *cobra.Command {
	return internal2.NewCommand("get-config [CONNECTOR_NAME]",
		internal2.WithAliases("getconfig", "getconf", "gc", "get", "g"),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithValidArgs("stripe"),
		internal2.WithShortDescription("Read a connector config"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			connectorConfig, _, err := client.PaymentsApi.ReadConnectorConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return internal2.WrapError(err, "reading connector config")
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
				internal2.Error(cmd.ErrOrStderr(), "Connection unknown.")
			}
			return nil
		}),
	)
}
