package config

import (
	"fmt"

	fctl "github.com/numary/fctl/pkg"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func ConfigModule() fx.Option {
	return fx.Module(
		"config",
		fx.Provide(fx.Annotate(
			NewConfig,
			fx.ParamTags(`group:"config-commands"`),
			fx.ResultTags(`group:"root-commands"`),
		)),
		fx.Provide(fx.Annotate(NewDebug, fx.ResultTags(`group:"config-commands"`))),
	)
}

func NewConfig(commands ...*cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use: "config",
	}

	command.AddCommand(commands...)

	return command
}

func NewDebug(
	config *fctl.Config,
	getProfile fctl.GetCurrentProfile,
) *cobra.Command {
	command := &cobra.Command{
		Use: "debug",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, name, err := getProfile()

			if err != nil {
				return err
			}

			fmt.Printf("current profile: %s\n", name)

			fmt.Println("available profiles")
			for key := range config.Profiles {
				fmt.Printf("* %s\n", key)
			}

			return nil
		},
	}

	return command
}
