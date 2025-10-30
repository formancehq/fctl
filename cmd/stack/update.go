package stack

import (
	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	nameFlag = "name"
)

type UpdateStore struct {
	Stack *membershipclient.Stack
}

type UpdateController struct {
	store   *UpdateStore
	profile fctl.Profile
}

var _ fctl.Controller[*UpdateStore] = (*UpdateController)(nil)

func NewDefaultStackUpdateStore() *UpdateStore {
	return &UpdateStore{
		Stack: &membershipclient.Stack{},
	}
}
func NewStackUpdateController() *UpdateController {
	return &UpdateController{
		store: NewDefaultStackUpdateStore(),
	}
}

func NewUpdateCommand() *cobra.Command {
	return fctl.NewMembershipCommand("update <stack-id>",
		fctl.WithShortDescription("Update a created stack, name, or metadata"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithStringFlag(nameFlag, "", "Name of the stack"),
		fctl.WithController(NewStackUpdateController()),
	)
}
func (c *UpdateController) GetStore() *UpdateStore {
	return c.store
}

func (c *UpdateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	store, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	c.profile = *profile

	stack, res, err := store.DefaultAPI.GetStack(cmd.Context(), organizationID, args[0]).Execute()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving stack")
	}
	if res.StatusCode > 300 {
		return nil, errors.New("stack not found")
	}

	name := fctl.GetString(cmd, nameFlag)
	if name == "" {
		name = stack.Data.Name
	}

	req := membershipclient.UpdateStackRequest{
		Name: name,
	}

	stackResponse, _, err := store.DefaultAPI.
		UpdateStack(cmd.Context(), organizationID, args[0]).
		UpdateStackRequest(req).
		Execute()
	if err != nil {
		return nil, errors.Wrap(err, "updating stack")
	}

	c.store.Stack = stackResponse.Data

	return c, nil
}

func (c *UpdateController) Render(cmd *cobra.Command, _ []string) error {
	return internal.PrintStackInformation(cmd.OutOrStdout(), c.store.Stack, nil)
}
