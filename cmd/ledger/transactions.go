package ledger

import (
	"fmt"

	fctl "github.com/numary/fctl/pkg"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func TransactionsModule() fx.Option {
	return fx.Module(
		"transactions",
		fx.Provide(fx.Annotate(
			NewTransactions,
			fx.ParamTags(`group:"transactions-commands"`),
			fx.ResultTags(`group:"ledger-commands"`),
		)),
		fx.Provide(fx.Annotate(NewTransactionsList, fx.ResultTags(`group:"transactions-commands"`))),
	)
}

func NewTransactions(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "transactions",
	}

	command.AddCommand(commands...)

	return command
}

func NewTransactionsList(client *fctl.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("list of transactions")
			ledger, _ := cmd.Flags().GetString("ledger")

			cursor, _, _ := client.Ledger.TransactionsApi.ListTransactions(cmd.Context(), ledger).Execute()

			for _, transaction := range cursor.Cursor.Data {
				fmt.Println(transaction)
			}
		},
	}

	cmd.Flags().String("address", "", "filter transactions by account address pattern")

	return cmd
}
