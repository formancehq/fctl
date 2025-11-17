package runs

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
)

type Show struct {
	components.Run
}

type ShowCtrl struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowCtrl)(nil)

func newShowStore() *Show {
	return &Show{
		Run: components.Run{},
	}
}

func NewShowCtrl() *ShowCtrl {
	return &ShowCtrl{
		store: newShowStore(),
	}
}

func NewShow() *cobra.Command {
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show run"),
		fctl.WithStringFlag("id", "", "Run ID"),
		fctl.WithController(NewShowCtrl()),
	)
}

func (c *ShowCtrl) GetStore() *Show {
	return c.store
}

func (c *ShowCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	app, err := store.Cli.ReadRun(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store.Run = app.RunResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.DefaultSection.Println("Run")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("Created At: %s", c.store.Run.CreatedAt)},
		{Level: 0, Text: fmt.Sprintf("Id: %s", c.store.Run.ID)},
		{Level: 0, Text: fmt.Sprintf("Configuration Id: %s", c.store.Run.ConfigurationVersion.ID)},
		{Level: 0, Text: fmt.Sprintf("Status: %s", c.store.Run.Status)},
		{Level: 0, Text: fmt.Sprintf("Message: %s", c.store.Run.Message)},
	}

	if err := pterm.
		DefaultBulletList.
		WithItems(items).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}
	return nil
}
