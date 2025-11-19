package apps

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
)

type Create struct {
	components.App
}

type CreateCtrl struct {
	store *Create
}

var _ fctl.Controller[*Create] = (*CreateCtrl)(nil)

func newCreateStore() *Create {
	return &Create{
		App: components.App{},
	}
}

func NewCreateCtrl() *CreateCtrl {
	return &CreateCtrl{
		store: newCreateStore(),
	}
}

func NewCreate() *cobra.Command {
	return fctl.NewCommand("create",
		fctl.WithShortDescription("Create apps"),
		fctl.WithController(NewCreateCtrl()),
	)
}

func (c *CreateCtrl) GetStore() *Create {
	return c.store
}

func (c *CreateCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}
	apps, err := apiClient.CreateApp(cmd.Context(), components.CreateAppRequest{
		OrganizationID: organizationID,
	})
	if err != nil {
		return nil, err
	}

	c.store.App = apps.AppResponse.Data

	return c, nil
}

func (c *CreateCtrl) Render(cmd *cobra.Command, args []string) error {
	if err := pterm.
		DefaultTable.
		WithHasHeader().
		WithData([][]string{
			{"ID", "Name"},
			{c.store.App.ID, c.store.App.Name},
		}).
		Render(); err != nil {
		return err
	}
	return nil
}
