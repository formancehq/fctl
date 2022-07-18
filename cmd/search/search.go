package search

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func SearchModule() fx.Option {
	return fx.Module(
		"seach",
		fx.Provide(fx.Annotate(
			NewSearch,
			fx.ParamTags(`group:"search-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
	)
}

func NewSearch(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "search",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("searching for something")
		},
	}

	return command
}
