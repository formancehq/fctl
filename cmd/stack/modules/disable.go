package modules

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

type DisableStore struct{}
type DisableController struct {
	store *DisableStore
}

var _ fctl.Controller[*DisableStore] = (*DisableController)(nil)

func NewDefaultDisableStore() *DisableStore {
	return &DisableStore{}
}

func NewDisableController() *DisableController {
	return &DisableController{
		store: NewDefaultDisableStore(),
	}
}

func NewDisableCommand() *cobra.Command {
	return fctl.NewStackCommand("disable <module-name>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("disable a module"),
		fctl.WithAliases("dis", "d"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewDisableController()),
	)
}
func (c *DisableController) GetStore() *DisableStore {
	return c.store
}

func (c *DisableController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	mbStackStore := fctl.GetMembershipStackStore(cmd.Context())

	_, err := mbStackStore.Client().DisableModule(cmd.Context(), mbStackStore.OrganizationId(), mbStackStore.StackId()).Name(args[0]).Execute()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DisableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Module disabled.")
	return nil
}
