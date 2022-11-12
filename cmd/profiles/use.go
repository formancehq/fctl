package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func ProfileNamesAutoCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := cmdutils.BindFlags(cmd); err != nil {
		return []string{}, 0
	}
	ret, err := config.ListProfiles(cmd.Context(), toComplete)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	return ret, cobra.ShellCompDirectiveDefault
}

func NewUseCommand() *cobra.Command {
	return cmdbuilder.NewCommand("use",
		cmdbuilder.WithAliases("u"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithShortDescription("Use profile"),
		cmdbuilder.WithValidArgsFunction(ProfileNamesAutoCompletion),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			config, err := config.Get(cmd.Context())
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
