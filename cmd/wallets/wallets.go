package wallets

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func WalletsModule() fx.Option {
	return fx.Module(
		"wallets",
		fx.Provide(fx.Annotate(
			NewWallets,
			fx.ParamTags(`group:"wallets-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewCreate, fx.ResultTags(`group:"wallets-commands"`))),
	)
}

func NewWallets(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "wallets",
	}

	command.AddCommand(commands...)

	return command
}

func NewCreate() *cobra.Command {
	return &cobra.Command{
		Use: "create",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("create wallet")
		},
	}
}
