package cmd

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showTransactionCommand = &cobra.Command{
	Use:   "show [TXID]",
	Short: "print a transaction",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
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
		printTransaction(cmd, rsp.Data)
		return nil
	},
}

func init() {
	transactionsCommand.AddCommand(showTransactionCommand)
}
