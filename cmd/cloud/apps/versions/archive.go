package versions

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type Archive []byte

type ArchiveCtrl struct {
	store Archive
}

var _ fctl.Controller[Archive] = (*ArchiveCtrl)(nil)

func newArchiveStore() Archive {
	return []byte{}
}

func NewArchiveCtrl() *ArchiveCtrl {
	return &ArchiveCtrl{
		store: newArchiveStore(),
	}
}

func NewArchive() *cobra.Command {
	return fctl.NewCommand("show-archive",
		fctl.WithShortDescription("Archive versions for an app"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithController(NewArchiveCtrl()),
	)
}

func (c *ArchiveCtrl) GetStore() Archive {
	return c.store
}

func (c *ArchiveCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	versions, err := store.Cli.ReadVersion(cmd.Context(), id, operations.WithAcceptHeaderOverride(operations.AcceptHeaderEnumApplicationGzip))
	if err != nil {
		return nil, err
	}
	defer versions.TwoHundredApplicationGzipResponseStream.Close()

	data, err := io.ReadAll(versions.TwoHundredApplicationGzipResponseStream)
	if err != nil {
		return nil, err
	}
	c.store = data
	return c, nil
}

func (c *ArchiveCtrl) Render(cmd *cobra.Command, args []string) error {
	fmt.Println(string(c.store))
	return nil
}
