package ledger

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewServerInfoCommand() *cobra.Command {
	return internal.NewCommand("server-infos",
		internal.WithArgs(cobra.ExactArgs(0)),
		internal.WithAliases("si"),
		internal.WithShortDescription("Read server info"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := ledgerClient.ServerApi.GetInfo(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("Server"), fmt.Sprint(response.Data.Server)})
			tableData = append(tableData, []string{pterm.LightCyan("Version"), fmt.Sprint(response.Data.Version)})
			tableData = append(tableData, []string{pterm.LightCyan("Storage driver"), fmt.Sprint(response.Data.Config.Storage.Driver)})

			if err := pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render(); err != nil {
				return err
			}

			internal.Highlightln(cmd.OutOrStdout(), "Ledgers :")
			if err := pterm.DefaultBulletList.
				WithWriter(cmd.OutOrStdout()).
				WithItems(internal.Map(response.Data.Config.Storage.Ledgers, func(ledger string) pterm.BulletListItem {
					return pterm.BulletListItem{
						Text:        ledger,
						TextStyle:   pterm.NewStyle(pterm.FgDefault),
						BulletStyle: pterm.NewStyle(pterm.FgLightCyan),
					}
				})).
				Render(); err != nil {
				return err
			}

			return nil
		}),
	)
}
