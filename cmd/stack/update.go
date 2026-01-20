package stack

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

const (
	nameFlag = "name"
)

type UpdateStore struct {
	Stack *components.Stack
}

type UpdateController struct {
	store   *UpdateStore
	profile fctl.Profile
}

var _ fctl.Controller[*UpdateStore] = (*UpdateController)(nil)

func NewDefaultStackUpdateStore() *UpdateStore {
	return &UpdateStore{
		Stack: &components.Stack{},
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	c.profile = *profile

	getRequest := operations.GetStackRequest{
		OrganizationID: organizationID,
		StackID:        args[0],
	}
	stackResponse, err := apiClient.GetStack(cmd.Context(), getRequest)
	if err != nil {
		return nil, fmt.Errorf("retrieving stack: %w", err)
	}
	if stackResponse.GetHTTPMeta().Response.StatusCode > 300 {
		return nil, errors.New("stack not found")
	}

	if stackResponse.ReadStackResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	stackData := stackResponse.ReadStackResponse.GetData()

	name := fctl.GetString(cmd, nameFlag)
	if name == "" {
		name = stackData.GetName()
	}

	updateData := components.StackData{
		Name: name,
	}

	updateRequest := operations.UpdateStackRequest{
		OrganizationID: organizationID,
		StackID:        args[0],
		Body:           &updateData,
	}

	updatedStackResponse, err := apiClient.UpdateStack(cmd.Context(), updateRequest)
	if err != nil {
		return nil, fmt.Errorf("updating stack: %w", err)
	}

	if updatedStackResponse.ReadStackResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Stack = updatedStackResponse.ReadStackResponse.GetData()

	return c, nil
}

func (c *UpdateController) Render(cmd *cobra.Command, _ []string) error {
	return internal.PrintStackInformation(cmd.OutOrStdout(), c.store.Stack, nil)
}
