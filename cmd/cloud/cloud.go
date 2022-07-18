package cloud

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const (
	CloudURI = "https://api.formance.cloud"
)

func CloudModule() fx.Option {
	return fx.Module(
		"cloud",
		fx.Provide(fx.Annotate(
			NewCloud,
			fx.ParamTags(`group:"cloud-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewApiKey, fx.ResultTags(`group:"cloud-commands"`))),
		fx.Provide(fx.Annotate(NewLogin, fx.ResultTags(`group:"cloud-commands"`))),
		fx.Provide(fx.Annotate(NewDeployment, fx.ResultTags(`group:"cloud-commands"`))),
	)
}

func NewCloud(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "cloud",
	}

	command.AddCommand(commands...)

	return command
}
