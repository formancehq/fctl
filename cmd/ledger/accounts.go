package ledger

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func AccountsModule() fx.Option {
	return fx.Module(
		"accounts",
		fx.Provide(fx.Annotate(
			NewAccounts,
			fx.ParamTags(`group:"accounts-commands"`),
			fx.ResultTags(`group:"ledger-commands"`),
		)),
		fx.Provide(fx.Annotate(NewAccountsList, fx.ResultTags(`group:"accounts-commands"`))),
		fx.Provide(fx.Annotate(NewAccountsGet, fx.ResultTags(`group:"accounts-commands"`))),
		fx.Provide(fx.Annotate(NewAccountSetMeta, fx.ResultTags(`group:"accounts-commands"`))),
	)
}

func NewAccounts(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "accounts",
	}

	command.AddCommand(commands...)

	return command
}

func NewAccountsList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			ledger, _ := cmd.Flags().GetString("ledger")
			fmt.Printf("list of accounts for ledger %s\n", ledger)
		},
	}
}

func NewAccountsGet() *cobra.Command {
	return &cobra.Command{
		Use:  "get",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("get account %s\n", args[0])
		},
	}
}

func NewAccountSetMeta() *cobra.Command {
	return &cobra.Command{
		Use:  "set-meta [account] [key:value]",
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("set meta of account %s\n", args[0])
		},
	}
}
