package accounts

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show [ADDRESS]",
		cmdbuilder.WithShortDescription("Show account"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithAliases("sh", "s"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			ledger := cmdutils.Viper(cmd.Context()).GetString(internal.LedgerFlag)
			rsp, _, err := ledgerClient.AccountsApi.GetAccount(cmd.Context(), ledger, args[0]).Execute()
			if err != nil {
				return err
			}

			if rsp.Data.Volumes != nil {
				tableData := pterm.TableData{}
				tableData = append(tableData, []string{"", "Input", "Output"})
				for asset, volumes := range *rsp.Data.Volumes {
					input := volumes["input"]
					output := volumes["output"]
					tableData = append(tableData, []string{pterm.LightCyan(asset), fmt.Sprint(input), fmt.Sprint(output)})
				}
				if err := pterm.DefaultTable.
					WithHasHeader(true).
					WithWriter(cmd.OutOrStdout()).
					WithData(tableData).
					Render(); err != nil {
					return err
				}
			}

			fmt.Fprintln(cmd.OutOrStdout())

			if err := internal.PrintMetadata(cmd.OutOrStdout(), *rsp.Data.Metadata); err != nil {
				return err
			}

			return nil
		}),
	)
}
