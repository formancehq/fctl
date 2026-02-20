package oauth_clients

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Show struct {
	Client components.OrganizationClient `json:"organizationClient"`
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	clientID := args[0]
	if clientID == "" {
		return nil, ErrMissingClientID
	}

	request := operations.OrganizationClientReadRequest{
		OrganizationID: organizationID,
		ClientID:       clientID,
	}

	response, err := apiClient.OrganizationClientRead(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ReadOrganizationClientResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Client = response.ReadOrganizationClientResponse.GetData()

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	return showOrganizationClient(cmd.OutOrStdout(), c.store.Client)
}
