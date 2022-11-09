package cmd

import (
	"strconv"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLedgerTransactionsShowCommand() *cobra.Command {
	return newCommand("show [TXID]",
		withShortDescription("print a transaction"),
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			config, err := getConfig()
			if err != nil {
				return err
			}

			ledgerClient, err := newLedgerClient(cmd, config)
			if err != nil {
				return err
			}

			txId, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return errors.Wrapf(err, "parsing txid")
			}

			ledger := viper.GetString(ledgerFlag)
			rsp, _, err := ledgerClient.TransactionsApi.GetTransaction(cmd.Context(), ledger, int32(txId)).Execute()
			if err != nil {
				return errors.Wrapf(err, "retrieving transaction")
			}

			internal.PrintLedgerTransaction(cmd.OutOrStdout(), rsp.Data)
			return nil
		}),
	)
}
