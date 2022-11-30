package ledger

import (
	"fmt"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewStatsCommand() *cobra.Command {
	return internal2.NewCommand("stats",
		internal2.WithArgs(cobra.ExactArgs(0)),
		internal2.WithAliases("st"),
		internal2.WithShortDescription("Read ledger stats"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}
			ledgerClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := ledgerClient.StatsApi.ReadStats(cmd.Context(), internal2.GetString(cmd, internal.LedgerFlag)).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("Transactions"), fmt.Sprint(response.Data.Transactions)})
			tableData = append(tableData, []string{pterm.LightCyan("Accounts"), fmt.Sprint(response.Data.Accounts)})

			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
