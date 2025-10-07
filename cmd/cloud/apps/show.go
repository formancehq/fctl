package apps

import (
	"fmt"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type Show struct {
	components.App
}

type ShowCtrl struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowCtrl)(nil)

func newShowStore() *Show {
	return &Show{}
}

func NewShowCtrl() *ShowCtrl {
	return &ShowCtrl{
		store: newShowStore(),
	}
}

func NewShow() *cobra.Command {
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show apps"),
		fctl.WithStringFlag("id", "", "App ID"),
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
	app, err := store.Cli.ReadApp(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store.App = app.AppResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.DefaultSection.Println("App")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("ID: %s", c.store.App.ID)},
		{Level: 0, Text: fmt.Sprintf("Name: %s", c.store.App.Name)},
		{Level: 0, Text: fmt.Sprintf("Run Status: %s", func() string {
			if c.store.App.CurrentRun == nil {
				return "N/A"
			}
			return c.store.App.CurrentRun.Status
		}())},
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
