package ledger

import (
	"fmt"

	fctl "github.com/numary/fctl/pkg"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func LedgerModule() fx.Option {
	return fx.Module(
		"ledger",
		fx.Provide(fx.Annotate(
			NewLedger,
			fx.ParamTags(`group:"ledger-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewList, fx.ResultTags(`group:"ledger-commands"`))),
		fx.Provide(fx.Annotate(NewCreate, fx.ResultTags(`group:"ledger-commands"`))),
		AccountsModule(),
		TransactionsModule(),
	)
}

func NewLedger(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "ledger",
	}

	command.PersistentFlags().StringP("ledger", "l", "", "ledger name")

	command.AddCommand(commands...)

	return command
}

func NewList(client *fctl.Client) *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("list of ledgers")
			info, _, _ := client.Ledger.ServerApi.GetInfo(cmd.Context()).Execute()

			for _, ledger := range info.Data.Config.Storage.Ledgers {
				fmt.Println(ledger)
			}
		},
	}
}

func NewCreate(client *fctl.Client) *cobra.Command {
	return &cobra.Command{
		Use:  "create",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, _, err := client.Ledger.TransactionsApi.ListTransactions(cmd.Context(), args[0]).Execute()

			if err != nil {
				fmt.Println(err)
			}

			fmt.Printf("ledger %s \n created", args[0])
		},
	}
}
