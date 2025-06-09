package oauth

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type Show struct {
	Organization *membershipclient.CreateClientResponse `json:"organization"`
}
type ShowController struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowController)(nil)

func NewDefaultShow() *Show {
	return &Show{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShow(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand(`show`,
		fctl.WithShortDescription("Show organization client"),
		fctl.WithController(NewShowController()),
	)
}

func (c *ShowController) GetStore() *Show {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	store := fctl.GetMembershipStore(cmd.Context())
	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	response, _, err := store.Client().ReadOrganizationClient(cmd.Context(), organizationID).Execute()
	if err != nil {
		return nil, err
	}

	c.store.Organization = response

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", c.store.Organization.Data.Id)},
		{"Client Last Digits", c.store.Organization.Data.Secret.LastDigits},
	}
	pterm.DefaultTable.WithHasHeader().WithData(data).Render()

	return nil
}
