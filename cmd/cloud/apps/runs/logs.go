package runs

// import (
// 	"fmt"

// 	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
// 	fctl "github.com/formancehq/fctl/pkg"
// 	"github.com/pterm/pterm"
// 	"github.com/spf13/cobra"
// )

// type Logs struct {
// 	components.LogsRunsResponseData
// }

// type LogsCtrl struct {
// 	store *Logs
// }

// var _ fctl.Controller[*Logs] = (*LogsCtrl)(nil)

// func newDefaultStore() *Logs {
// 	return &Logs{
// 		LogsRunsResponseData: components.LogsRunsResponseData{},
// 	}
// }

// func NewLogsCtrl() *LogsCtrl {
// 	return &LogsCtrl{
// 		store: newDefaultStore(),
// 	}
// }

// func NewLogs() *cobra.Command {
// 	return fctl.NewCommand("Logs",
// 		fctl.WithAliases("ls"),
// 		fctl.WithShortDescription("Logs runs for an app"),
// 		fctl.WithStringFlag("id", "", "App ID"),
// 		fctl.WithController(NewLogsCtrl()),
// 	)
// }

// func (c *LogsCtrl) GetStore() *Logs {
// 	return c.store
// }

// func (c *LogsCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
// 	store := fctl.GetDeployServerStore(cmd.Context())
// 	id := fctl.GetString(cmd, "id")
// 	if id == "" {
// 		return nil, fmt.Errorf("id is required")
// 	}
// 	runs, err := store.Cli.ReadAppRuns(cmd.Context(), id, nil, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	c.store.LogsRunsResponseData = runs.LogsRunsResponse.Data

// 	return c, nil
// }

// func (c *LogsCtrl) Render(cmd *cobra.Command, args []string) error {
// 	data := [][]string{
// 		{"Created At", "ID", "Configuration Id", "Status", "Message"},
// 	}

// 	for _, run := range c.store.Items {
// 		data = append(data, []string{
// 			run.CreatedAt.String(),
// 			run.ID,
// 			string(run.ConfigurationVersion.ID),
// 			string(run.Status),
// 			run.Message,
// 		})
// 	}
// 	if err := pterm.
// 		DefaultTable.
// 		WithHasHeader().
// 		WithWriter(cmd.OutOrStdout()).
// 		WithData(data).
// 		Render(); err != nil {
// 		return err
// 	}
// 	return nil
// }
