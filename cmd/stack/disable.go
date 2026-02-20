package stack

import (
	"errors"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type DisabledStore struct {
	Stack  *components.Stack `json:"stack"`
	Status string            `json:"status"`
}
type DisableController struct {
	store *DisabledStore
}

var _ fctl.Controller[*DisabledStore] = (*DisableController)(nil)

func NewDisableStore() *DisabledStore {
	return &DisabledStore{
		Stack:  &components.Stack{},
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	var stack *components.Stack
	if len(args) == 1 {
		if fctl.GetString(cmd, stackNameFlag) != "" {
			return nil, errors.New("need either an id of a name specified using --name flag")
		}

		getRequest := operations.GetStackRequest{
			OrganizationID: organizationID,
			StackID:        args[0],
		}
		rsp, err := apiClient.GetStack(cmd.Context(), getRequest)
		if err != nil {
			return nil, err
		}
		if rsp.ReadStackResponse == nil {
			return nil, fmt.Errorf("unexpected response: no data")
		}
		stack = rsp.ReadStackResponse.GetData()
	} else {
		if fctl.GetString(cmd, stackNameFlag) == "" {
			return nil, errors.New("need either an id of a name specified using --name flag")
		}
		listRequest := operations.ListStacksRequest{
			OrganizationID: organizationID,
		}
		stacksResponse, err := apiClient.ListStacks(cmd.Context(), listRequest)
		if err != nil {
			return nil, fmt.Errorf("listing stacks: %w", err)
		}
		if stacksResponse.ListStacksResponse == nil {
			return nil, fmt.Errorf("unexpected response: no data")
		}
		for _, s := range stacksResponse.ListStacksResponse.GetData() {
			if s.GetName() == fctl.GetString(cmd, stackNameFlag) {
				stackData := s
				stack = &stackData
				break
			}
		}
	}
	if stack == nil {
		return nil, errors.New("Stack not found")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to disable stack '%s'", stack.GetName()) {
		return nil, fctl.ErrMissingApproval
	}

	disableRequest := operations.DisableStackRequest{
		OrganizationID: organizationID,
		StackID:        stack.GetID(),
	}
	if _, err := apiClient.DisableStack(cmd.Context(), disableRequest); err != nil {
		return nil, fmt.Errorf("stack disable: %w", err)
	}

	c.store.Stack = stack
	c.store.Status = "OK"

	return c, nil
}

func (c *DisableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack disabled.")
	return nil
}
