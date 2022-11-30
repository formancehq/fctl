package accounts

import (
	"fmt"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return internal2.NewCommand("show [ADDRESS]",
		internal2.WithShortDescription("Show account"),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithAliases("sh", "s"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			ledgerClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			ledger := internal2.GetString(cmd, internal.LedgerFlag)
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

			if err := internal.PrintMetadata(cmd.OutOrStdout(), rsp.Data.Metadata); err != nil {
				return err
			}

			return nil
		}),
	)
}
