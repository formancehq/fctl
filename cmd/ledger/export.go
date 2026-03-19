package ledger

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	"github.com/formancehq/fctl/v3/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ExportStore struct {
	response *http.Response
}
type ExportController struct {
	store          *ExportStore
	outputFileFlag string
}

var _ fctl.Controller[*ExportStore] = (*ExportController)(nil)

func NewDefaultExportStore() *ExportStore {
	return &ExportStore{}
}

func NewExportController() *ExportController {
	return &ExportController{
		store:          NewDefaultExportStore(),
		outputFileFlag: "file",
	}
}

func NewExportCommand() *cobra.Command {
	c := NewExportController()
	return fctl.NewCommand("export",
		fctl.WithShortDescription("Export a ledger"),
		fctl.WithStringFlag(c.outputFileFlag, "", "Export to file"),
		fctl.WithController[*ExportStore](c),
	)
}

func (c *ExportController) GetStore() *ExportStore {
	return c.store
}

func (c *ExportController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	ctx := cmd.Context()
	out := fctl.GetString(cmd, "file")
	if out != "" {
		ctx = context.WithValue(ctx, "path", out)
	}

	ret, err := stackClient.Ledger.V2.ExportLogs(ctx, operations.V2ExportLogsRequest{
		Ledger: fctl.GetString(cmd, internal.LedgerFlag),
	})
	if err != nil {
		return nil, err
	}

	c.store.response = ret.RawResponse

	return c, nil
}

func (c *ExportController) Render(cmd *cobra.Command, _ []string) error {
	outFile := fctl.GetString(cmd, "file")
	var out io.Writer
	if outFile == "" {
		out = os.Stdout
		_, err := io.Copy(out, c.store.response.Body)
		return err
	}
	return nil
}
