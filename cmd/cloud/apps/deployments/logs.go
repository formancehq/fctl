package deployments

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	"github.com/formancehq/fctl/v3/cmd/cloud/apps/printer"
	fctl "github.com/formancehq/fctl/v3/pkg"
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
		fctl.WithShortDescription("Read logs for a deployment"),
		fctl.WithStringFlag("id", "", "Deployment ID"),
		fctl.WithController(NewLogsCtrl()),
	)
}

func (c *LogsCtrl) GetStore() Logs {
	return c.store
}

func (c *LogsCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}

	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	logs, err := apiClient.ReadDeploymentLogs(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store = Logs(logs.ReadLogsResponse.Data)

	return c, nil
}

func (c *LogsCtrl) Render(cmd *cobra.Command, _ []string) error {
	return printer.RenderLogs(cmd.OutOrStdout(), c.store)
}
