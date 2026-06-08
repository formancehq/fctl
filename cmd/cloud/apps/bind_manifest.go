package apps

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type BindManifest struct {
	AppID      string `json:"appId"`
	ManifestID string `json:"manifestId"`
}

type BindManifestCtrl struct {
	store *BindManifest
}

var _ fctl.Controller[*BindManifest] = (*BindManifestCtrl)(nil)

func NewBindManifestCtrl() *BindManifestCtrl {
	return &BindManifestCtrl{store: &BindManifest{}}
}

func NewBindManifest() *cobra.Command {
	return fctl.NewCommand("bind-manifest",
		fctl.WithShortDescription("Bind a manifest to an app"),
		fctl.WithStringFlag("app-id", "", "App ID"),
		fctl.WithStringFlag("manifest-id", "", "Manifest ID to bind"),
		fctl.WithController(NewBindManifestCtrl()),
	)
}

func (c *BindManifestCtrl) GetStore() *BindManifest { return c.store }

func (c *BindManifestCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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

	appID := fctl.GetString(cmd, "app-id")
	if appID == "" {
		return nil, fmt.Errorf("app-id is required")
	}
	manifestID := fctl.GetString(cmd, "manifest-id")
	if manifestID == "" {
		return nil, fmt.Errorf("manifest-id is required")
	}

	if _, err := apiClient.AttachAppManifest(cmd.Context(), appID, components.AttachManifestRequest{
		ManifestID: manifestID,
	}); err != nil {
		return nil, err
	}

	c.store.AppID = appID
	c.store.ManifestID = manifestID
	return c, nil
}

func (c *BindManifestCtrl) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Printfln("Manifest %s bound to app %s", c.store.ManifestID, c.store.AppID)
	return nil
}
