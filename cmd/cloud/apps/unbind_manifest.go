package apps

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UnbindManifest struct {
	AppID string `json:"appId"`
}

type UnbindManifestCtrl struct {
	store *UnbindManifest
}

var _ fctl.Controller[*UnbindManifest] = (*UnbindManifestCtrl)(nil)

func NewUnbindManifestCtrl() *UnbindManifestCtrl {
	return &UnbindManifestCtrl{store: &UnbindManifest{}}
}

func NewUnbindManifest() *cobra.Command {
	return fctl.NewCommand("unbind-manifest",
		fctl.WithShortDescription("Unbind the manifest from an app"),
		fctl.WithStringFlag("app-id", "", "App ID"),
		fctl.WithController(NewUnbindManifestCtrl()),
	)
}

func (c *UnbindManifestCtrl) GetStore() *UnbindManifest { return c.store }

func (c *UnbindManifestCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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

	if _, err := apiClient.DetachAppManifest(cmd.Context(), appID); err != nil {
		return nil, err
	}

	c.store.AppID = appID
	return c, nil
}

func (c *UnbindManifestCtrl) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Printfln("Manifest unbound from app %s", c.store.AppID)
	return nil
}
