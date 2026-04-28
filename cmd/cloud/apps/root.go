package apps

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/cloud/apps/runs"
	"github.com/formancehq/fctl/v3/cmd/cloud/apps/variables"
	"github.com/formancehq/fctl/v3/cmd/cloud/apps/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	cmd := fctl.NewMembershipCommand("apps",
		fctl.WithShortDescription("* New * Apps manifests management"),
		fctl.WithPersistentBoolFlag("experimental", false, "Enable experimental commands"),
		fctl.WithPersistentStringFlag(fctl.FrameworkURIFlag, "https://deploy.formance.cloud", "Framework URI"),
		fctl.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			ok, err := cmd.Flags().GetBool("experimental")
			if err != nil {
				return err
			}

			if !ok {
				return fmt.Errorf("the apps command is experimental, please use the --experimental flag to enable it")
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
			runs.NewCommand(),
			versions.NewCommand(),
			variables.NewCommand(),
		),
	)

	return cmd
}
