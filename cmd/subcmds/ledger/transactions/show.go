package transactions

import (
	"strconv"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLedgerTransactionsShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show [TXID]",
		cmdbuilder.WithShortDescription("print a transaction"),
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

			txId, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return errors.Wrapf(err, "parsing txid")
			}

			ledger := viper.GetString(internal.LedgerFlag)
			rsp, _, err := ledgerClient.TransactionsApi.GetTransaction(cmd.Context(), ledger, int32(txId)).Execute()
			if err != nil {
				return errors.Wrapf(err, "retrieving transaction")
			}

			internal.PrintLedgerTransaction(cmd.OutOrStdout(), rsp.Data)
			return nil
		}),
	)
}
