package oauth

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type Create struct {
	Organization *membershipclient.CreateClientResponse `json:"organization"`
}
type CreateController struct {
	store *Create
}

var _ fctl.Controller[*Create] = (*CreateController)(nil)

func NewDefaultCreate() *Create {
	return &Create{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultCreate(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand(`create`,
		fctl.WithShortDescription("Create organization client"),
		fctl.WithConfirmFlag(),
		fctl.WithController(NewCreateController()),
	)
}

func (c *CreateController) GetStore() *Create {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	store := fctl.GetMembershipStore(cmd.Context())
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to create a new organization oauth client") {
		return nil, fctl.ErrMissingApproval
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	response, _, err := store.Client().CreateOrganizationClient(cmd.Context(), organizationID).Execute()
	if err != nil {
		return nil, err
	}

	c.store.Organization = response

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", c.store.Organization.Data.Id)},
		{"Client Secret", *c.store.Organization.Data.Secret.Clear},
	}
	pterm.DefaultTable.WithHasHeader().WithData(data).Render()

	return nil
}
