package stack

import (
	"errors"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type EnableStore struct {
	Stack  *components.Stack `json:"stack"`
	Status string            `json:"status"`
}
type EnableController struct {
	store *EnableStore
}

var _ fctl.Controller[*EnableStore] = (*EnableController)(nil)

func NewEnableStore() *EnableStore {
	return &EnableStore{
		Stack:  &components.Stack{},
		Status: "",
	}
}

func NewEnableController() *EnableController {
	return &EnableController{
		store: NewEnableStore(),
	}
}

func NewEnableCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)
	return fctl.NewMembershipCommand("enable (<stack-id> | --name=<stack-name>)",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Enable a stack"),
		fctl.WithArgs(cobra.MaximumNArgs(1)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithStringFlag(stackNameFlag, "", "Stack to enable"),
		fctl.WithController(NewEnableController()),
	)
}
func (c *EnableController) GetStore() *EnableStore {
	return c.store
}

func (c *EnableController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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
			return nil, errors.New("need either an id or a name specified using --name flag")
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

	if !fctl.CheckStackApprobation(cmd, "You are about to enable stack '%s'", stack.GetName()) {
		return nil, fctl.ErrMissingApproval
	}

	enableRequest := operations.EnableStackRequest{
		OrganizationID: organizationID,
		StackID:        stack.GetID(),
	}
	if _, err := apiClient.EnableStack(cmd.Context(), enableRequest); err != nil {
		return nil, fmt.Errorf("stack enable: %w", err)
	}

	c.store.Stack = stack
	c.store.Status = "OK"

	return c, nil
}

func (c *EnableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack enabled.")
	return nil
}
