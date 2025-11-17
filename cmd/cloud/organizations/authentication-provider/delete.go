package authentication_provider

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

type Delete struct{}
type DeleteController struct {
	store *Delete
}

var _ fctl.Controller[*Delete] = (*DeleteController)(nil)

func NewDefaultDelete() *Delete {
	return &Delete{}
}

func NewDeleteController() *DeleteController {
	return &DeleteController{
		store: NewDefaultDelete(),
	}
}

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand(`delete`,
		fctl.WithShortDescription("Delete authorization provider of organization"),
		fctl.WithController(NewDeleteController()),
	)
}

func (c *DeleteController) GetStore() *Delete {
	return c.store
}

func (c *DeleteController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	store := fctl.GetMembershipStore(cmd.Context())
	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	_, err = store.Client().
		DeleteAuthenticationProvider(cmd.Context(), organizationID).
		Execute()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DeleteController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Authorization provider deleted successfully")

	return nil
}
