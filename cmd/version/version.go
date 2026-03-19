package version

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

var (
	Version   = "develop"
	Commit    = "-"
	BuildDate = "-"
)

type Store struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	Commit    string `json:"commit"`
}
type Controller struct {
	store *Store
}

var _ fctl.Controller[*Store] = (*Controller)(nil)

func NewDefaultVersionStore() *Store {
	return &Store{
		Version:   Version,
		BuildDate: BuildDate,
		Commit:    Commit,
	}
}

func NewVersionController() *Controller {
	return &Controller{
		store: NewDefaultVersionStore(),
	}
}

func NewCommand() *cobra.Command {
	return fctl.NewCommand("version",
		fctl.WithShortDescription("Get version"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*Store](NewVersionController()),
	)
}

func (c *Controller) GetStore() *Store {
	return c.store
}

func (c *Controller) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	return c, nil
}

// TODO: This need to use the ui.NewListModel
func (c *Controller) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Version"), c.store.Version})
	tableData = append(tableData, []string{pterm.LightCyan("Date"), c.store.BuildDate})
	tableData = append(tableData, []string{pterm.LightCyan("Commit"), c.store.Commit})
	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
