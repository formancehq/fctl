package oauth_clients

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to delete an OAuth client") {
		return nil, fctl.ErrMissingApproval
	}

	clientID := args[0]
	if clientID == "" {
		return nil, ErrMissingClientID
	}

	request := operations.OrganizationClientDeleteRequest{
		OrganizationID: organizationID,
		ClientID:       clientID,
	}

	_, err = apiClient.OrganizationClientDelete(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Println("Organization client deleted successfully")
	return nil
}
