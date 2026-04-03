package deployments

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type DeleteStore struct {
	Name string
}

type DeleteCtrl struct {
	store *DeleteStore
}

var _ fctl.Controller[*DeleteStore] = (*DeleteCtrl)(nil)

func newDeleteStore() *DeleteStore {
	return &DeleteStore{}
}

func NewDeleteCtrl() *DeleteCtrl {
	return &DeleteCtrl{
		store: newDeleteStore(),
	}
}

func NewDelete() *cobra.Command {
	return fctl.NewCommand("delete",
		fctl.WithShortDescription("Delete a deployment"),
		fctl.WithStringFlag("id", "", "App ID (required)"),
		fctl.WithStringFlag("name", "", "Deployment name (required)"),
		fctl.WithController(NewDeleteCtrl()),
	)
}

func (c *DeleteCtrl) GetStore() *DeleteStore {
	return c.store
}

func (c *DeleteCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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

	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("--id is required")
	}
	name := fctl.GetString(cmd, "name")
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}

	cmd.SilenceUsage = true

	_, err = apiClient.DeleteDeployment(cmd.Context(), id, name)
	if err != nil {
		return nil, err
	}

	c.store.Name = name

	return c, nil
}

func (c *DeleteCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Println("Deployment deleted", c.store.Name)
	return nil
}
