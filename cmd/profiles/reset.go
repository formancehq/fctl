package profiles

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ResetStore struct {
	Success bool `json:"success"`
}
type ResetController struct {
	store *ResetStore
}

var _ fctl.Controller[*ResetStore] = (*ResetController)(nil)

func NewResetStore() *ResetStore {
	return &ResetStore{
		Success: false,
	}
}

func NewResetController() *ResetController {
	return &ResetController{
		store: NewResetStore(),
	}
}

func (c *ResetController) GetStore() *ResetStore {
	return c.store
}

func (c *ResetController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if err := fctl.ResetProfile(cmd, args[0]); err != nil {
		return nil, err
	}

	c.store.Success = true

	return c, nil
}

func (c *ResetController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Profile reset on default !")
	return nil
}

func NewResetCommand() *cobra.Command {
	return fctl.NewCommand("reset <name>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithShortDescription("Reset a profile keeping the environment"),
		fctl.WithValidArgsFunction(ProfileNamesAutoCompletion),
		fctl.WithController(NewResetController()),
	)
}
