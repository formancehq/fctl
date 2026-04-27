package manifests

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Create struct {
	ID      string
	Name    string
	Version int64
}

type CreateCtrl struct {
	store *Create
}

var _ fctl.Controller[*Create] = (*CreateCtrl)(nil)

func newCreateStore() *Create {
	return &Create{}
}

func NewCreateCtrl() *CreateCtrl {
	return &CreateCtrl{
		store: newCreateStore(),
	}
}

func NewCreate() *cobra.Command {
	return fctl.NewCommand("create",
		fctl.WithShortDescription("Create a new manifest"),
		fctl.WithStringFlag("name", "", "Manifest name"),
		fctl.WithStringFlag("app-id", "", "Optional app ID to scope the manifest"),
		fctl.WithStringFlag("path", "", "Path to YAML manifest file"),
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

	path := fctl.GetString(cmd, "path")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var appID *string
	if id := fctl.GetString(cmd, "app-id"); id != "" {
		appID = &id
	}

	resp, err := apiClient.CreateManifestRaw(cmd.Context(), name, data, appID)
	if err != nil {
		return nil, err
	}

	c.store.ID = resp.CreateManifestResponse.Data.ID
	c.store.Name = resp.CreateManifestResponse.Data.Name
	c.store.Version = resp.CreateManifestResponse.Data.Version

	return c, nil
}

func (c *CreateCtrl) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.Printfln("Manifest created: %s (version %d)", c.store.ID, c.store.Version)
	return nil
}
