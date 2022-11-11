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
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			config.GetCurrentProfile(cfg).SetDefaultOrganization(args[0])

			return errors.Wrap(cfg.Persist(), "Updating config")
		}),
	)
}
