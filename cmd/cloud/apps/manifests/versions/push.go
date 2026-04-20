package versions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Push struct {
	ManifestID string
	Version    int64
}

type PushCtrl struct {
	store *Push
}

var _ fctl.Controller[*Push] = (*PushCtrl)(nil)

func newPushStore() *Push {
	return &Push{}
}

func NewPushCtrl() *PushCtrl {
	return &PushCtrl{
		store: newPushStore(),
	}
}

func NewPush() *cobra.Command {
	return fctl.NewCommand("push",
		fctl.WithShortDescription("Push a new version of a manifest"),
		fctl.WithStringFlag("manifest-id", "", "Manifest ID"),
		fctl.WithStringFlag("path", "", "Path to YAML manifest file"),
		fctl.WithController(NewPushCtrl()),
	)
}

func (c *PushCtrl) GetStore() *Push {
	return c.store
}

func (c *PushCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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

	path := fctl.GetString(cmd, "path")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	resp, err := apiClient.PushManifestVersionRaw(cmd.Context(), manifestID, data)
	if err != nil {
		return nil, err
	}

	c.store.ManifestID = resp.ManifestVersionResponse.Data.ManifestID
	c.store.Version = resp.ManifestVersionResponse.Data.Version

	return c, nil
}

func (c *PushCtrl) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.Printfln("Manifest version %d pushed for %s", c.store.Version, c.store.ManifestID)
	return nil
}
