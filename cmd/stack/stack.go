package stack

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func StackModule() fx.Option {
	return fx.Module(
		"stack",
		fx.Provide(fx.Annotate(
			NewStack,
			fx.ParamTags(`group:"stack-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewStatus, fx.ResultTags(`group:"stack-commands"`))),
	)
}

func NewStack(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "stack",
	}

	command.AddCommand(commands...)

	return command
}

func NewStatus() *cobra.Command {
	return &cobra.Command{
		Use: "status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("status of stack")
		},
	}
}
