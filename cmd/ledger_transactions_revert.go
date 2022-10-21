package cmd

import (
	"strconv"

	"github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLedgerTransactionsRevertCommand() *cobra.Command {
	return newCommand("revert [TXID]",
		withShortDescription("revert a transaction"),
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			ledgerClient, err := fctl.NewLedgerClientFromContext(cmd.Context())
			if err != nil {
				return err
			}

			txId, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return errors.Wrapf(err, "parsing txid")
			}

			ledger := viper.GetString(ledgerFlag)
			rsp, _, err := ledgerClient.TransactionsApi.RevertTransaction(cmd.Context(), ledger, int32(txId)).Execute()
			if err != nil {
				return errors.Wrapf(err, "reverting transaction")
			}

			fctl.PrintLedgerTransaction(cmd.OutOrStdout(), rsp.Data)
			return nil
		}),
	)
}
