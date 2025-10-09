package versions

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/internal/deployserverclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

type Manifest []byte

type ManifestCtrl struct {
	store Manifest
}

var _ fctl.Controller[Manifest] = (*ManifestCtrl)(nil)

func newManifestStore() []byte {
	return []byte{}
}

func NewManifestCtrl() *ManifestCtrl {
	return &ManifestCtrl{
		store: newManifestStore(),
	}
}

func NewManifest() *cobra.Command {
	return fctl.NewCommand("show-manifest",
		fctl.WithShortDescription("Manifest versions for an app"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithController(NewManifestCtrl()),
	)
}

func (c *ManifestCtrl) GetStore() Manifest {
	return c.store
}

func (c *ManifestCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	versions, err := store.Cli.ReadVersion(cmd.Context(), id, operations.WithAcceptHeaderOverride(operations.AcceptHeaderEnumApplicationYaml))
	if err != nil {
		return nil, err
	}
	defer versions.TwoHundredApplicationYamlResponseStream.Close()
	data, err := io.ReadAll(versions.TwoHundredApplicationYamlResponseStream)
	if err != nil {
		return nil, err
	}
	c.store = data
	return c, nil
}

func (c *ManifestCtrl) Render(cmd *cobra.Command, args []string) error {

	return nil
}
