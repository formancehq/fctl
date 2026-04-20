package manifests

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Show struct {
	components.Manifest
}

type ShowCtrl struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowCtrl)(nil)

func newShowStore() *Show {
	return &Show{}
}

func NewShowCtrl() *ShowCtrl {
	return &ShowCtrl{
		store: newShowStore(),
	}
}

func NewShow() *cobra.Command {
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show manifest details"),
		fctl.WithStringFlag("id", "", "Manifest ID"),
		fctl.WithController(NewShowCtrl()),
	)
}

func (c *ShowCtrl) GetStore() *Show {
	return c.store
}

func (c *ShowCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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

	manifest, err := apiClient.ReadManifest(cmd.Context(), id, nil)
	if err != nil {
		return nil, err
	}

	c.store.Manifest = manifest.ManifestResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, _ []string) error {
	pterm.DefaultSection.Println("Manifest")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("ID: %s", c.store.ID)},
		{Level: 0, Text: fmt.Sprintf("Name: %s", c.store.Name)},
		{Level: 0, Text: fmt.Sprintf("Latest Version: %d", c.store.LatestVersion)},
		{Level: 0, Text: fmt.Sprintf("Created At: %s", c.store.CreatedAt)},
		{Level: 0, Text: fmt.Sprintf("Updated At: %s", c.store.UpdatedAt)},
	}

	if c.store.AppID != nil {
		items = append(items, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("App ID: %s", *c.store.AppID)})
	}

	if err := pterm.
		DefaultBulletList.
		WithItems(items).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}

	return nil
}
