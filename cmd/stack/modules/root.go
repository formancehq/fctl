package modules

import (
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("modules",
		fctl.WithShortDescription("Manage your modules"),
		fctl.WithAliases("module", "mod"),
		fctl.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			orgStore := fctl.GetOrganizationStore(cmd)
			if err := orgStore.CheckRegionCapability(string(membershipclient.MODULE_LIST), func(s []any) bool {
				return len(s) > 0
			})(cmd, args); err != nil {
				return err
			}

			if err := fctl.CheckMembershipCapabilities(membershipclient.MODULE_SELECTION)(cmd, args); err != nil {
				return err
			}

			if err := fctl.NewMembershipStackStore(cmd); err != nil {
				return err
			}

			return nil
		}),
		fctl.WithChildCommands(
			NewDisableCommand(),
			NewEnableCommand(),
			NewListCommand(),
		),
	)
}
