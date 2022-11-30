package profiles

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewSetDefaultOrganizationCommand() *cobra.Command {
	return internal.NewCommand("set-default-organization",
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithAliases("sdo"),
		internal.WithShortDescription("Set default organization"),
		internal.WithValidArgsFunction(ProfileNamesAutoCompletion),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			internal.GetCurrentProfile(cmd, cfg).SetDefaultOrganization(args[0])

			if err := cfg.Persist(); err != nil {
				return errors.Wrap(err, "Updating config")
			}
			internal.Success(cmd.OutOrStdout(), "Default organization updated!")
			return nil
		}),
	)
}
