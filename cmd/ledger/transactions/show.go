package transactions

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return internal2.NewCommand("show [TXID]",
		internal2.WithShortDescription("Print a transaction"),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithAliases("sh"),
		internal2.WithValidArgs("last"),
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
			txId, err := internal.TransactionIDOrLastN(cmd.Context(), ledgerClient, ledger, args[0])
			if err != nil {
				return err
			}

			rsp, _, err := ledgerClient.TransactionsApi.GetTransaction(cmd.Context(), ledger, int32(txId)).Execute()
			if err != nil {
				return errors.Wrapf(err, "retrieving transaction")
			}

			return internal.PrintTransaction(cmd.OutOrStdout(), rsp.Data)
		}),
	)
}
