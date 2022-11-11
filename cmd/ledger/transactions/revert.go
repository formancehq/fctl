package transactions

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRevertCommand() *cobra.Command {
	return cmdbuilder.NewCommand("revert [TXID]",
		cmdbuilder.WithShortDescription("Revert a transaction"),
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

			ledger := viper.GetString(internal.LedgerFlag)
			txId, err := internal.TransactionIDOrLast(cmd.Context(), ledgerClient, ledger, args[0])
			if err != nil {
				return err
			}

			rsp, _, err := ledgerClient.TransactionsApi.RevertTransaction(cmd.Context(), ledger, int32(txId)).Execute()
			if err != nil {
				return errors.Wrapf(err, "reverting transaction")
			}

			return internal.PrintTransaction(cmd.OutOrStdout(), rsp.Data)
		}),
	)
}
