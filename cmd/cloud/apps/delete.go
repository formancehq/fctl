package apps

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
		fctl.WithShortDescription("Delete apps"),
		fctl.WithStringFlag("id", "", "App ID"),
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

	profile, err := fctl.LoadCurrentProfile(cmd, *cfg)
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

	store, err := fctl.NewAppDeployClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		fctl.GetCurrentProfileName(cmd, *cfg),
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
	_, err = store.DeleteApp(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store.ID = id

	return c, nil
}

func (c *DeleteCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Println("App deleted", c.store.ID)
	return nil
}
