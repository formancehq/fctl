package ledger

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewStatsCommand() *cobra.Command {
	return cmdbuilder.NewCommand("stats",
		cmdbuilder.WithArgs(cobra.ExactArgs(0)),
		cmdbuilder.WithAliases("st"),
		cmdbuilder.WithShortDescription("Read ledger stats"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := ledgerClient.StatsApi.ReadStats(cmd.Context(), viper.GetString(internal.LedgerFlag)).Execute()
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