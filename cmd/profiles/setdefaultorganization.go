package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewSetDefaultOrganizationCommand() *cobra.Command {
	return cmdbuilder.NewCommand("set-default-organization",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithAliases("sdo"),
		cmdbuilder.WithShortDescription("Set default organization"),
		cmdbuilder.WithValidArgsFunction(ProfileNamesAutoCompletion),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			config.GetCurrentProfile(cmd.Context(), cfg).SetDefaultOrganization(args[0])

			if err := cfg.Persist(); err != nil {
				return errors.Wrap(err, "Updating config")
			}
			cmdbuilder.Success(cmd.OutOrStdout(), "Default organization updated!")
			return nil
		}),
	)
}
