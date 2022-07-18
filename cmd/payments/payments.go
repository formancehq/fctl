package payments

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func PaymentsModule() fx.Option {
	return fx.Module(
		"payments",
		fx.Provide(fx.Annotate(
			NewPayments,
			fx.ParamTags(`group:"payments-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewListPayments, fx.ResultTags(`group:"payments-commands"`))),
		fx.Provide(fx.Annotate(NewConnectors, fx.ResultTags(`group:"payments-commands"`))),
	)
}

func NewPayments(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "payments",
	}

	command.AddCommand(commands...)

	return command
}

func NewListPayments() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("list of payments")
		},
	}
}
