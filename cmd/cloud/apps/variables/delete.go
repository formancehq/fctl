package variables

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

type Delete struct {
	ID string
}

type DeleteCtrl struct {
	store *Delete
}

var _ fctl.Controller[*Delete] = (*DeleteCtrl)(nil)

func newDeleteStore() *Delete {
	return &Delete{}
}

func NewDeleteCtrl() *DeleteCtrl {
	return &DeleteCtrl{
		store: newDeleteStore(),
	}
}

func NewDelete() *cobra.Command {
	return fctl.NewCommand("delete",
		fctl.WithShortDescription("Delete a variable"),
		fctl.WithStringFlag("id", "", "Variable id"),
		fctl.WithStringFlag("app-id", "", "App ID"),
		fctl.WithController(NewDeleteCtrl()),
	)
}

func (c *DeleteCtrl) GetStore() *Delete {
	return c.store
}

func (c *DeleteCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	relyingParty, err := fctl.GetAuthRelyingParty(cmd.Context(), fctl.GetHttpClient(cmd), profile.MembershipURI)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewAppDeployClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	appID := fctl.GetString(cmd, "app-id")
	if appID == "" {
		return nil, fmt.Errorf("app-id is required")
	}
	_, err = apiClient.DeleteAppVariable(cmd.Context(), appID, id)
	if err != nil {
		return nil, err
	}

	c.store.ID = id

	return c, nil
}

func (c *DeleteCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Println("Variable deleted", c.store.ID)
	return nil
}
