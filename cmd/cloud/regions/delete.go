package regions

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type DeleteStore struct {
	Success bool `json:"success"`
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
	return fctl.NewCommand("delete <region-id>",
		fctl.WithAliases("del", "d"),
		fctl.WithShortDescription("Delete a private region"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*DeleteStore](NewDeleteController()),
	)
}

func (c *DeleteController) GetStore() *DeleteStore {
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

	request := operations.DeleteRegionRequest{
		OrganizationID: organizationID,
		RegionID:       args[0],
	}

	_, err = apiClient.DeleteRegion(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.Success = true

	return c, nil
}

func (c *DeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Region deleted successfully!")

	return nil

}
