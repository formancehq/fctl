package stack

import (
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type DisabledStore struct {
	Stack  *membershipclient.Stack `json:"stack"`
	Status string                  `json:"status"`
}
type DisableController struct {
	store *DisabledStore
}

var _ fctl.Controller[*DisabledStore] = (*DisableController)(nil)

func NewDisableStore() *DisabledStore {
	return &DisabledStore{
		Stack:  &membershipclient.Stack{},
		Status: "",
	}
}

func NewDisableController() *DisableController {
	return &DisableController{
		store: NewDisableStore(),
	}
}

func NewDisableCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)
	return fctl.NewMembershipCommand("disable (<stack-id> | --name=<stack-name>)",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Disable a stack"),
		fctl.WithArgs(cobra.MaximumNArgs(1)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithStringFlag(stackNameFlag, "", "Stack to disable"),
		fctl.WithController(NewDisableController()),
	)
}
func (c *DisableController) GetStore() *DisabledStore {
	return c.store
}

func (c *DisableController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	const (
		stackNameFlag = "name"
	)

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
	var stack *membershipclient.Stack
	if len(args) == 1 {
		if fctl.GetString(cmd, stackNameFlag) != "" {
			return nil, errors.New("need either an id of a name specified using --name flag")
		}

		rsp, _, err := store.DefaultAPI.GetStack(cmd.Context(), organizationID, args[0]).Execute()
		if err != nil {
			return nil, err
		}
		stack = rsp.Data
	} else {
		if fctl.GetString(cmd, stackNameFlag) == "" {
			return nil, errors.New("need either an id of a name specified using --name flag")
		}
		stacks, _, err := store.DefaultAPI.ListStacks(cmd.Context(), organizationID).Execute()
		if err != nil {
			return nil, errors.Wrap(err, "listing stacks")
		}
		for _, s := range stacks.Data {
			if s.Name == fctl.GetString(cmd, stackNameFlag) {
				stack = &s
				break
			}
		}
	}
	if stack == nil {
		return nil, errors.New("Stack not found")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to disable stack '%s'", stack.Name) {
		return nil, fctl.ErrMissingApproval
	}

	if _, err := store.DefaultAPI.DisableStack(cmd.Context(), organizationID, stack.Id).Execute(); err != nil {
		return nil, errors.Wrap(err, "stack disable")
	}

	c.store.Stack = stack
	c.store.Status = "OK"

	return c, nil
}

func (c *DisableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack disabled.")
	return nil
}
