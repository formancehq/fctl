package transactions

import (
	"strconv"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	internal2 "github.com/formancehq/fctl/cmd/ledger/internal"
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

			ledgerClient, err := internal2.NewLedgerClient(cmd, cfg)
			if err != nil {
				return err
			}

			txId, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return errors.Wrapf(err, "parsing txid")
			}

			ledger := viper.GetString(internal2.LedgerFlag)
			rsp, _, err := ledgerClient.TransactionsApi.GetTransaction(cmd.Context(), ledger, int32(txId)).Execute()
			if err != nil {
				return errors.Wrapf(err, "retrieving transaction")
			}

			internal2.PrintLedgerTransaction(cmd.OutOrStdout(), rsp.Data)
			return nil
		}),
	)
}
