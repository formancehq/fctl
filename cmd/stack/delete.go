package stack

import (
	"errors"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

const (
	stackNameFlag = "name"
	forceFlag     = "force"
)

type DeletedStackStore struct {
	Stack  *components.Stack `json:"stack"`
	Status string            `json:"status"`
}
type StackDeleteController struct {
	store *DeletedStackStore
}

var _ fctl.Controller[*DeletedStackStore] = (*StackDeleteController)(nil)

func NewDefaultDeletedStackStore() *DeletedStackStore {
	return &DeletedStackStore{
		Stack:  &components.Stack{},
		Status: "",
	}
}

func NewStackDeleteController() *StackDeleteController {
	return &StackDeleteController{
		store: NewDefaultDeletedStackStore(),
	}
}

func NewDeleteCommand() *cobra.Command {
	return fctl.NewMembershipCommand("delete (<stack-id> | --name=<stack-name>)",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Delete a stack"),
		fctl.WithAliases("del", "d"),
		fctl.WithArgs(cobra.MaximumNArgs(1)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithStringFlag(stackNameFlag, "", "Stack to delete"),
		fctl.WithBoolFlag(forceFlag, false, "Force to delete a stack without retention policy"),
		fctl.WithController(NewStackDeleteController()),
	)
}
func (c *StackDeleteController) GetStore() *DeletedStackStore {
	return c.store
}

func (c *StackDeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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

	if !fctl.CheckStackApprobation(cmd, "You are about to delete stack '%s'", stack.GetName()) {
		return nil, fctl.ErrMissingApproval
	}

	deleteRequest := operations.DeleteStackRequest{
		OrganizationID: organizationID,
		StackID:        stack.GetID(),
	}
	if fctl.GetBool(cmd, forceFlag) {
		deleteRequest.Force = pointer.For(true)
	}

	_, err = apiClient.DeleteStack(cmd.Context(), deleteRequest)
	if err != nil {
		return nil, fmt.Errorf("deleting stack: %w", err)
	}

	c.store.Stack = stack
	c.store.Status = "OK"

	return c, nil
}

func (c *StackDeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack deleted.")
	return nil
}
