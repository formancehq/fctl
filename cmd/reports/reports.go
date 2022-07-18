package reports

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func ReportsModule() fx.Option {
	return fx.Module(
		"reports",
		fx.Provide(fx.Annotate(
			NewReports,
			fx.ParamTags(`group:"reports-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewList, fx.ResultTags(`group:"reports-commands"`))),
		fx.Provide(fx.Annotate(NewCreate, fx.ResultTags(`group:"reports-commands"`))),
	)
}

func NewReports(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "reports",
	}

	command.AddCommand(commands...)

	return command
}

func NewList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("list of reports")
		},
	}
}

func NewCreate() *cobra.Command {
	return &cobra.Command{
		Use: "create",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("create a report")
		},
	}
}
