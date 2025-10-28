package users

import (
	"fmt"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type UnlinkStore struct {
	Stack  *components.Stack `json:"stack"`
	Status string            `json:"status"`
}
type UnlinkController struct {
	store *UnlinkStore
}

var _ fctl.Controller[*UnlinkStore] = (*UnlinkController)(nil)

func NewDefaultUnlinkStore() *UnlinkStore {
	return &UnlinkStore{
		Stack:  &components.Stack{},
		Status: "",
	}
}

func NewUnlinkController() *UnlinkController {
	return &UnlinkController{
		store: NewDefaultUnlinkStore(),
	}
}

func NewUnlinkCommand() *cobra.Command {
	return fctl.NewMembershipCommand("unlink <user-id>",
		fctl.WithShortDescription("Unlink stack user within an organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewUnlinkController()),
	)
}
func (c *UnlinkController) GetStore() *UnlinkStore {
	return c.store
}

func (c *UnlinkController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
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

	request := operations.DeleteStackUserAccessRequest{
		OrganizationID: organizationID,
		StackID:        stackID,
		UserID:         args[0],
	}

	response, err := apiClient.DeleteStackUserAccess(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("error deleting stack user access: %s", response.Error.GetErrorCode())
	}

	return c, nil
}

func (c *UnlinkController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack user access deleted.")
	return nil
}
