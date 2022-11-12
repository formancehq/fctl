package profiles

import (
	"strings"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewUseCommand() *cobra.Command {
	return cmdbuilder.NewCommand("use",
		cmdbuilder.WithAliases("u"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithShortDescription("Use profile"),
		cmdbuilder.WithValidArgsFunction(func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			config, err := config.Get()
			if err != nil {
				return []string{}, cobra.ShellCompDirectiveError
			}

			ret := make([]string, 0)
			for p := range config.GetProfiles() {
				if strings.HasPrefix(p, toComplete) {
					ret = append(ret, p)
				}
			}
			return ret, cobra.ShellCompDirectiveDefault
		}),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			config, err := config.Get()
			if err != nil {
				return err
			}

			config.SetCurrentProfileName(args[0])
			if err := config.Persist(); err != nil {
				return errors.Wrap(err, "Updating config")
			}
			cmdbuilder.Success(cmd.OutOrStdout(), "Selected profile updated!")
			return nil
		}),
	)
}
