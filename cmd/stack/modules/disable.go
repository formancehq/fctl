package modules

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to disable a module") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.DisableModuleRequest{
		OrganizationID: organizationID,
		StackID:        stackID,
		Name:           args[0],
	}

	_, err = apiClient.DisableModule(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DisableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Module disabled.")
	return nil
}
