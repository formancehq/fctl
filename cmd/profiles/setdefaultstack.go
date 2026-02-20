package profiles

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
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

	currentProfile, profileName, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *currentProfile)
	if err != nil {
		return nil, err
	}

	if !currentProfile.RootTokens.ID.Claims.HasStackAccess(organizationID, args[0]) {
		return nil, fmt.Errorf("stack %s not found in your access list", args[0])
	}

	currentProfile.DefaultStack = args[0]
	if err := fctl.WriteProfile(cmd, profileName, *currentProfile); err != nil {
		return nil, fmt.Errorf("Updating config: %w", err)
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
