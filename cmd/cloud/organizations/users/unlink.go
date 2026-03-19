package users

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UnlinkStore struct {
	OrganizationID string `json:"organizationId"`
	UserID         string `json:"userId"`
}
type DeleteController struct {
	store *UnlinkStore
}

var _ fctl.Controller[*UnlinkStore] = (*DeleteController)(nil)

func NewDefaultUnlinkStore() *UnlinkStore {
	return &UnlinkStore{}
}

func NewUnlinkController() *DeleteController {
	return &DeleteController{
		store: NewDefaultUnlinkStore(),
	}
}

func NewUnlinkCommand() *cobra.Command {
	return fctl.NewCommand("unlink <user-id>",
		fctl.WithAliases("u", "un"),
		fctl.WithShortDescription("Unlink user from organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewUnlinkController()),
	)
}

func (c *DeleteController) GetStore() *UnlinkStore {
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

	request := operations.DeleteUserFromOrganizationRequest{
		OrganizationID: organizationID,
		UserID:         args[0],
	}

	_, err = apiClient.DeleteUserFromOrganization(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.OrganizationID = organizationID
	c.store.UserID = args[0]

	return c, nil
}

func (c *DeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("User '%s' Deleted from organization '%s'", c.store.UserID, c.store.OrganizationID)

	return nil

}
