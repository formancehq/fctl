package apps

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
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
		fctl.WithStringFlag("name", "", "App name"),
		fctl.WithStringFlag("stack-id", "", "Optional existing stack ID to claim"),
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

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}

	name := fctl.GetString(cmd, "name")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	req := components.CreateAppRequest{
		Name: name,
	}
	if stackID := fctl.GetString(cmd, "stack-id"); stackID != "" {
		req.StackID = &stackID
	}

	apps, err := apiClient.CreateApp(cmd.Context(), req)
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
