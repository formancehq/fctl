package profiles

import (
	"errors"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

type SetDefaultOrganizationStore struct {
	Success bool `json:"success"`
}
type SetDefaultOrganizationController struct {
	store *SetDefaultOrganizationStore
}

var _ fctl.Controller[*SetDefaultOrganizationStore] = (*SetDefaultOrganizationController)(nil)

func NewDefaultProfilesSetDefaultOrganizationStore() *SetDefaultOrganizationStore {
	return &SetDefaultOrganizationStore{
		Success: false,
	}
}

func NewProfilesSetDefaultOrganizationController() *SetDefaultOrganizationController {
	return &SetDefaultOrganizationController{
		store: NewDefaultProfilesSetDefaultOrganizationStore(),
	}
}

func (c *SetDefaultOrganizationController) GetStore() *SetDefaultOrganizationStore {
	return c.store
}

func (c *SetDefaultOrganizationController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	currentProfile, profileName, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	if !currentProfile.IsConnected() {
		return nil, errors.New("You are not connected, please run 'fctl login'")
	}

	if !currentProfile.RootTokens.ID.Claims.HasOrganizationAccess(args[0]) {
		return nil, fmt.Errorf("organization %s not found in your access list", args[0])
	}

	currentProfile.DefaultOrganization = args[0]
	currentProfile.DefaultStack = ""

	if err := fctl.WriteProfile(cmd, profileName, *currentProfile); err != nil {
		return nil, fmt.Errorf("Updating config: %w", err)
	}

	c.store.Success = true
	return c, nil
}

func (c *SetDefaultOrganizationController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Default organization updated!")
	return nil
}

func NewSetDefaultOrganizationCommand() *cobra.Command {
	return fctl.NewCommand("set-default-organization <organization-id>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithAliases("sdo"),
		fctl.WithShortDescription("Set default organization"),
		fctl.WithValidArgsFunction(fctl.OrganizationCompletion),
		fctl.WithController(NewProfilesSetDefaultOrganizationController()),
	)
}
