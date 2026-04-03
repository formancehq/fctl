package apps

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/apps/deployments"
	"github.com/formancehq/fctl/v3/cmd/apps/manifest"
	"github.com/formancehq/fctl/v3/cmd/apps/runs"
	"github.com/formancehq/fctl/v3/cmd/apps/variables"
	"github.com/formancehq/fctl/v3/cmd/apps/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	cmd := fctl.NewMembershipCommand("apps",
		fctl.WithShortDescription("Deploy Formance applications from manifest files"),
		fctl.WithPersistentStringFlag(fctl.FrameworkURIFlag, "https://deploy.formance.cloud", "Framework URI"),
		fctl.WithAliases("app"),
		fctl.WithChildCommands(
			NewInit(),
			NewList(),
			NewCreate(),
			NewDelete(),
			NewShow(),
			NewDeploy(),
			manifest.NewCommand(),
			deployments.NewCommand(),
			runs.NewCommand(),
			versions.NewCommand(),
			variables.NewCommand(),
		),
	)

	return cmd
}
