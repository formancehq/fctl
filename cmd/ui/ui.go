package ui

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func UIModule() fx.Option {
	return fx.Module(
		"ui",
		fx.Provide(fx.Annotate(
			NewUI,
			fx.ParamTags(`group:"ui-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewStart, fx.ResultTags(`group:"ui-commands"`))),
	)
}

func NewUI(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "ui",
	}

	command.AddCommand(commands...)

	return command
}

func NewStart() *cobra.Command {
	return &cobra.Command{
		Use: "start",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("start ui")
		},
	}
}
