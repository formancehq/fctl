package organizations

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type DeleteStore struct {
	OrganizationId string `json:"organizationId"`
	Success        bool   `json:"success"`
}
type DeleteController struct {
	store *DeleteStore
}

var _ fctl.Controller[*DeleteStore] = (*DeleteController)(nil)

func NewDefaultDeleteStore() *DeleteStore {
	return &DeleteStore{}
}

func NewDeleteController() *DeleteController {
	return &DeleteController{
		store: NewDefaultDeleteStore(),
	}
}

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete <organization-id>",
		fctl.WithAliases("del", "d"),
		fctl.WithShortDescription("Delete organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithController(NewDeleteController()),
	)
}

func (c *DeleteController) GetStore() *DeleteStore {
	return c.store
}

func (c *DeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetMembershipStore(cmd.Context())

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to delete an organization") {
		return nil, fctl.ErrMissingApproval
	}

	_, err := store.Client().DeleteOrganization(cmd.Context(), args[0]).
		Execute()
	if err != nil {
		return nil, err
	}

	c.store.OrganizationId = args[0]
	c.store.Success = true

	return c, nil
}

func (c *DeleteController) Render(cmd *cobra.Command, args []string) error {

	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Organization '%s' deleted", c.store.OrganizationId)

	return nil
}
