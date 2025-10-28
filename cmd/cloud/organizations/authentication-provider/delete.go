package authentication_provider

import (
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
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
	return fctl.NewCommand(`delete`,
		fctl.WithShortDescription("Delete authorization provider of organization"),
		fctl.WithController(NewDeleteController()),
	)
}

func (c *DeleteController) GetStore() *Delete {
	return c.store
}

func (c *DeleteController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	request := operations.DeleteAuthenticationProviderRequest{
		OrganizationID: organizationID,
	}

	_, err = apiClient.DeleteAuthenticationProvider(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DeleteController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Authorization provider deleted successfully")

	return nil
}
