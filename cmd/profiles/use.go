package profiles

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func ProfileNamesAutoCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ret, err := internal.ListProfiles(cmd, toComplete)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	return ret, cobra.ShellCompDirectiveDefault
}

func NewUseCommand() *cobra.Command {
	return internal.NewCommand("use",
		internal.WithAliases("u"),
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithShortDescription("Use profile"),
		internal.WithValidArgsFunction(ProfileNamesAutoCompletion),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			config, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			config.SetCurrentProfileName(args[0])
			if err := config.Persist(); err != nil {
				return errors.Wrap(err, "Updating config")
			}
			internal.Success(cmd.OutOrStdout(), "Selected profile updated!")
			return nil
		}),
	)
}
