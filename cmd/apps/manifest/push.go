package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Push struct {
	components.ManifestVersion
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
		fctl.WithShortDescription("Push a new manifest version"),
		fctl.WithStringFlag("id", "", "App ID (required)"),
		fctl.WithStringFlag("path", "", "Path to the manifest YAML file (required)"),
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

	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("--id is required")
	}
	path := fctl.GetString(cmd, "path")
	if path == "" {
		return nil, fmt.Errorf("--path is required")
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	cmd.SilenceUsage = true

	res, err := apiClient.PushManifest(cmd.Context(), id, data)
	if err != nil {
		return nil, err
	}

	c.store.ManifestVersion = res.ManifestVersionResponse.Data

	return c, nil
}

func (c *PushCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Printfln("Manifest pushed successfully")
	if err := pterm.
		DefaultTable.
		WithHasHeader().
		WithData([][]string{
			{"Version", "App ID", "Created At"},
			{strconv.Itoa(c.store.ManifestVersion.Version), c.store.ManifestVersion.AppID, c.store.ManifestVersion.CreatedAt.String()},
		}).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}
	return nil
}
