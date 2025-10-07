package apps

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := fctl.NewMembershipCommand("apps",
		fctl.WithShortDescription("* New * Apps manifests management"),
		fctl.WithPersistentBoolFlag("experimental", false, "Enable experimental commands"),
		fctl.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			ok, err := cmd.Flags().GetBool("experimental")
			if err != nil {
				return err
			}

			if !ok {
				return fmt.Errorf("the apps command is experimental, please use the --experimental flag to enable it")
			}

			if err := fctl.NewDeployServerStore(cmd); err != nil {
				return err
			}

			return nil
		}),
		fctl.WithAliases("app"),
		fctl.WithChildCommands(
			NewList(),
			NewCreate(),
			NewDelete(),
			NewShow(),
			NewDeploy(),
			// runs.NewCommand(),
			// variables.NewCommand(),
		),
	)

	return cmd
}
