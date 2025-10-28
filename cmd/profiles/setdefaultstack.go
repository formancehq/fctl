package profiles

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type SetDefaultStackStore struct {
	Success bool `json:"success"`
}
type SetDefaultStackController struct {
	store *SetDefaultStackStore
}

var _ fctl.Controller[*SetDefaultStackStore] = (*SetDefaultStackController)(nil)

func NewDefaultSetDefaultStackStore() *SetDefaultStackStore {
	return &SetDefaultStackStore{
		Success: false,
	}
}

func NewSetDefaultStackController() *SetDefaultStackController {
	return &SetDefaultStackController{
		store: NewDefaultSetDefaultStackStore(),
	}
}

func (c *SetDefaultStackController) GetStore() *SetDefaultStackStore {
	return c.store
}

func (c *SetDefaultStackController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}
	
	// todo: check if stack exists in id token

	currentProfile, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	// todo: check existence using id token
	currentProfile.DefaultStack = args[0]
	if err := fctl.WriteProfile(cmd, cfg.CurrentProfile, *currentProfile); err != nil {
		return nil, errors.Wrap(err, "Updating config")
	}

	c.store.Success = true
	return c, nil
}

func (c *SetDefaultStackController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Default stack updated!")
	return nil
}

func NewSetDefaultStackCommand() *cobra.Command {
	return fctl.NewCommand("set-default-stack <stack-id>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithAliases("sds"),
		fctl.WithShortDescription("Set default stack"),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithController(NewSetDefaultStackController()),
	)
}
