package runs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/cloud/apps/printer"
	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
)

type Logs []components.Log
type LogsCtrl struct {
	store Logs
}

var _ fctl.Controller[Logs] = (*LogsCtrl)(nil)

func newLogStore() Logs {
	return Logs{}
}

func NewLogsCtrl() *LogsCtrl {
	return &LogsCtrl{
		store: newLogStore(),
	}
}

func NewLogs() *cobra.Command {
	return fctl.NewCommand("logs",
		fctl.WithAliases("ls"),
		fctl.WithShortDescription("Read logs related to an app run"),
		fctl.WithStringFlag("id", "", "run ID"),
		fctl.WithController(NewLogsCtrl()),
	)
}

func (c *LogsCtrl) GetStore() Logs {
	return c.store
}

func (c *LogsCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewAppDeployClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	logs, err := apiClient.ReadRunLogs(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	s := Logs(logs.ReadLogsResponse.Data)
	c.store = s

	return c, nil
}

func (c *LogsCtrl) Render(cmd *cobra.Command, args []string) error {
	return printer.RenderLogs(cmd.OutOrStdout(), c.store)
}
