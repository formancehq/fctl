package connectors

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewGetConfigCommand() *cobra.Command {
	return fctl.NewCommand("get-config [CONNECTOR_NAME]",
		fctl.WithAliases("getconfig", "getconf", "gc", "get", "g"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgs("stripe"),
		fctl.WithShortDescription("Read a connector config"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			organizationID, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := fctl.ResolveStack(cmd, cfg, organizationID)
			if err != nil {
				return err
			}

			client, err := fctl.NewStackClient(cmd, cfg, stack)
			if err != nil {
				return err
			}

			connectorConfig, _, err := client.PaymentsApi.ReadConnectorConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return fctl.WrapError(err, "reading connector config")
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
				fctl.Error(cmd.ErrOrStderr(), "Connection unknown.")
			}
			return nil
		}),
	)
}
