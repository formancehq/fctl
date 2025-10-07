package runs

import (
	"fmt"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type List struct {
	components.ListRunsResponseData
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ListRunsResponseData: components.ListRunsResponseData{},
	}
}

func NewListCtrl() *ListCtrl {
	return &ListCtrl{
		store: newDefaultStore(),
	}
}

func NewList() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls"),
		fctl.WithShortDescription("List runs for an app"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithController(NewListCtrl()),
	)
}

func (c *ListCtrl) GetStore() *List {
	return c.store
}

func (c *ListCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	runs, err := store.Cli.ReadAppRuns(cmd.Context(), id, nil, nil)
	if err != nil {
		return nil, err
	}

	c.store.ListRunsResponseData = runs.ListRunsResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Created At", "ID", "Configuration Id", "Status", "Message"},
	}

	for _, run := range c.store.Items {
		data = append(data, []string{
			run.CreatedAt.String(),
			run.ID,
			string(run.ConfigurationVersion.ID),
			string(run.Status),
			run.Message,
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
