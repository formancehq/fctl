package runs

import (
	"fmt"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	logs, err := store.Cli.ReadRunLogs(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	s := Logs(logs.ReadLogsResponse.Data)
	c.store = s

	return c, nil
}

func (c *LogsCtrl) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Timestamp", "Summary", "Details"},
	}

	for _, log := range c.store {
		data = append(data, []string{
			log.Timestamp.String(),
			log.Diagnostic.Summary,
			log.Diagnostic.Detail,
		})
	}
	if err := pterm.
		DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(data).
		Render(); err != nil {
		return err
	}
	return nil
}
