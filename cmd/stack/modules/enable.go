package modules

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
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
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewEnableController()),
	)
}
func (c *EnableController) GetStore() *EnableStore {
	return c.store
}

func (c *EnableController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	_, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to enable a module") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.EnableModuleRequest{
		OrganizationID: organizationID,
		StackID:        stackID,
		Name:           args[0],
	}

	_, err = apiClient.EnableModule(cmd.Context(), request)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *EnableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Module enabled")
	return nil
}
