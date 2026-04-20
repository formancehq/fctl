package versions

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Show struct {
	components.ManifestVersion
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
		fctl.WithShortDescription("Show a specific manifest version"),
		fctl.WithStringFlag("manifest-id", "", "Manifest ID"),
		fctl.WithStringFlag("version", "latest", "Version number or 'latest'"),
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

	manifestID := fctl.GetString(cmd, "manifest-id")
	if manifestID == "" {
		return nil, fmt.Errorf("manifest-id is required")
	}

	version := fctl.GetString(cmd, "version")

	resp, err := apiClient.ReadManifestVersion(cmd.Context(), manifestID, version)
	if err != nil {
		return nil, err
	}

	c.store.ManifestVersion = resp.ManifestVersionResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, _ []string) error {
	pterm.DefaultSection.Println("Manifest Version")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("Manifest ID: %s", c.store.ManifestID)},
		{Level: 0, Text: fmt.Sprintf("Version: %s", strconv.FormatInt(c.store.Version, 10))},
		{Level: 0, Text: fmt.Sprintf("Created At: %s", c.store.CreatedAt)},
	}

	if c.store.Content != nil {
		items = append(items, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("Content: %s", *c.store.Content)})
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
