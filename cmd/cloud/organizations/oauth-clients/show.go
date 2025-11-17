package oauth_clients

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
)

type Show struct {
	Client membershipclient.OrganizationClient `json:"organizationClient"`
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
	return fctl.NewCommand(`show <client_id>`,
		fctl.WithShortDescription("Show organization OAuth client"),
		fctl.WithController(NewShowController()),
		fctl.WithArgs(cobra.ExactArgs(1)),
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

	clientID := args[0]
	if clientID == "" {
		return nil, ErrMissingClientID
	}

	response, _, err := store.Client().OrganizationClientRead(cmd.Context(), organizationID, clientID).Execute()
	if err != nil {
		return nil, err
	}

	c.store.Client = response.Data

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	return showOrganizationClient(cmd.OutOrStdout(), c.store.Client)
}
