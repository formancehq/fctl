package accounts

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	internal2 "github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLedgerAccountsListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithShortDescription("List accounts"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get()
			if err != nil {
				return err
			}

			ledgerClient, err := internal2.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			ledger := viper.GetString(internal2.LedgerFlag)
			rsp, _, err := ledgerClient.AccountsApi.ListAccounts(cmd.Context(), ledger).Execute()
			if err != nil {
				return err
			}
			if len(rsp.Cursor.Data) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No accounts found.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Accounts: ")
			for _, account := range rsp.Cursor.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "-> Account: %s\r\n", account.Address)
				if account.Metadata != nil && len(*account.Metadata) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Metadata:")
					for k, v := range *account.Metadata {
						fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", k, v)
					}
				}
			}
			return nil
		}),
	)
}
