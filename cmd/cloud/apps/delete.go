package apps

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

type Delete struct {
	ID string
}

type DeleteCtrl struct {
	store *Delete
}

var _ fctl.Controller[*Delete] = (*DeleteCtrl)(nil)

func newDeleteStore() *Delete {
	return &Delete{}
}

func NewDeleteCtrl() *DeleteCtrl {
	return &DeleteCtrl{
		store: newDeleteStore(),
	}
}

func NewDelete() *cobra.Command {
	return fctl.NewCommand("delete",
		fctl.WithShortDescription("Delete apps"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithController(NewDeleteCtrl()),
	)
}

func (c *DeleteCtrl) GetStore() *Delete {
	return c.store
}

func (c *DeleteCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	_, err := store.Cli.DeleteApp(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store.ID = id

	return c, nil
}

func (c *DeleteCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Println("App deleted", c.store.ID)
	return nil
}
