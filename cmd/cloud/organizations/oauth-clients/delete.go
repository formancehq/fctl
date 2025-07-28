package oauth

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
	return fctl.NewCommand(`delete <client_id>`,
		fctl.WithShortDescription("Delete organization OAuth client"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController(NewDeleteController()),
	)
}

func (c *DeleteController) GetStore() *Delete {
	return c.store
}

func (c *DeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetMembershipStore(cmd.Context())
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to delete a new organization OAuth client") {
		return nil, fctl.ErrMissingApproval
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	clientID := args[0]
	if clientID == "" {
		return nil, ErrMissingClientID
	}

	_, err = store.Client().OrganizationClientDelete(cmd.Context(), organizationID, clientID).Execute()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Println("Organization client deleted successfully")
	return nil
}
