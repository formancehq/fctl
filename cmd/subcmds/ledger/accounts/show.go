package accounts

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/internal"
	ledgerclient "github.com/numary/ledger/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func printAccount(cmd *cobra.Command, account ledgerclient.AccountWithVolumesAndBalances) {
	if account.Volumes != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "Volumes:")
		for asset, v := range *account.Volumes {
			fmt.Fprintf(cmd.OutOrStdout(), "\t\tAsset: %s\t\tInput: %d\tOutput: %d\tBalance: %d\r\n",
				asset, v["input"], v["output"], v["balance"])
		}
	}

	if account.Metadata != nil && len(*account.Metadata) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Metadata:")
		for k, v := range *account.Metadata {
			fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", k, v)
		}
	}
}

func NewLedgerAccountsShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show [ADDRESS]",
		cmdbuilder.WithShortDescription("display account"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			ledger := viper.GetString(internal.LedgerFlag)
			rsp, _, err := ledgerClient.AccountsApi.GetAccount(cmd.Context(), ledger, args[0]).Execute()
			if err != nil {
				return err
			}

			printAccount(cmd, rsp.Data)
			return nil
		}),
	)
}
