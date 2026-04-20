package manifests

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Update struct {
	components.Manifest
}

type UpdateCtrl struct {
	store *Update
}

var _ fctl.Controller[*Update] = (*UpdateCtrl)(nil)

func newUpdateStore() *Update {
	return &Update{}
}

func NewUpdateCtrl() *UpdateCtrl {
	return &UpdateCtrl{
		store: newUpdateStore(),
	}
}

func NewUpdate() *cobra.Command {
	return fctl.NewCommand("update",
		fctl.WithShortDescription("Update manifest metadata"),
		fctl.WithStringFlag("id", "", "Manifest ID"),
		fctl.WithStringFlag("name", "", "New name for the manifest"),
		fctl.WithController(NewUpdateCtrl()),
	)
}

func (c *UpdateCtrl) GetStore() *Update {
	return c.store
}

func (c *UpdateCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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
		return nil, fmt.Errorf("id is required")
	}

	name := fctl.GetString(cmd, "name")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	resp, err := apiClient.UpdateManifest(cmd.Context(), id, components.UpdateManifestRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	c.store.Manifest = resp.ManifestResponse.Data

	return c, nil
}

func (c *UpdateCtrl) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.Printfln("Manifest updated: %s", c.store.ID)
	return nil
}
