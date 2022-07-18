package reconciliation

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func ReconciliationModule() fx.Option {
	return fx.Module(
		"reconciliation",
		fx.Provide(fx.Annotate(
			NewReconciliation,
			fx.ParamTags(`group:"reconciliation-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
	)
}

func NewReconciliation(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "reconciliation",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("reconciling")
		},
	}

	return command
}
