package transactions

import (
	"strconv"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLedgerTransactionsShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show [TXID]",
		cmdbuilder.WithShortDescription("Print a transaction"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			txId, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return errors.Wrapf(err, "parsing txid")
			}

			ledger := viper.GetString(internal.LedgerFlag)
			rsp, _, err := ledgerClient.TransactionsApi.GetTransaction(cmd.Context(), ledger, int32(txId)).Execute()
			if err != nil {
				return errors.Wrapf(err, "retrieving transaction")
			}

			return internal.PrintTransaction(cmd.OutOrStdout(), rsp.Data)
		}),
	)
}
