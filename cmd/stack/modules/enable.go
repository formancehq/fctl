package modules

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type EnableStore struct {
}
type EnableController struct {
	store *EnableStore
}

var _ fctl.Controller[*EnableStore] = (*EnableController)(nil)

func NewDefaultEnableStore() *EnableStore {
	return &EnableStore{}
}

func NewEnableController() *EnableController {
	return &EnableController{
		store: NewDefaultEnableStore(),
	}
}

func NewEnableCommand() *cobra.Command {
	return fctl.NewStackCommand("enable <module-name>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Enable a module"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController(NewEnableController()),
	)
}
func (c *EnableController) GetStore() *EnableStore {
	return c.store
}

func (c *EnableController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	mbStackStore := fctl.GetMembershipStackStore(cmd.Context())
	_, err := mbStackStore.Client().EnableModule(cmd.Context(), mbStackStore.OrganizationId(), mbStackStore.StackId()).Name(args[0]).Execute()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *EnableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Module enabled")
	return nil
}
