package stack

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type StackRestoreStore struct {
	Stack    *components.Stack           `json:"stack"`
	Versions *shared.GetVersionsResponse `json:"versions"`
}
type StackRestoreController struct {
	store  *StackRestoreStore
	config fctl.Config
}

var _ fctl.Controller[*StackRestoreStore] = (*StackRestoreController)(nil)

func NewDefaultVersionStore() *StackRestoreStore {
	return &StackRestoreStore{
		Stack:    &components.Stack{},
		Versions: &shared.GetVersionsResponse{},
	}
}

func NewStackRestoreController() *StackRestoreController {
	return &StackRestoreController{
		store: NewDefaultVersionStore(),
	}
}

func NewRestoreStackCommand() *cobra.Command {
	const stackNameFlag = "name"

	return fctl.NewMembershipCommand("restore <stack-id>",
		fctl.WithShortDescription("Restore a deleted stack"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithStringFlag(stackNameFlag, "", ""),
		fctl.WithController(NewStackRestoreController()),
	)
}
func (c *StackRestoreController) GetStore() *StackRestoreStore {
	return c.store
}

func (c *StackRestoreController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	cfg, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	var stack *components.Stack
	if len(args) == 1 {
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
	}

	if stack == nil {
		return nil, errors.New("Stack not found")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to restore stack '%s'", stack.GetName()) {
		return nil, fctl.ErrMissingApproval
	}

	restoreRequest := operations.RestoreStackRequest{
		OrganizationID: organizationID,
		StackID:        args[0],
	}

	response, err := apiClient.RestoreStack(cmd.Context(), restoreRequest)
	if err != nil {
		return nil, err
	}

	if response.ReadStackResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	restoredStackData := response.ReadStackResponse.GetData()

	if !fctl.GetBool(cmd, nowaitFlag) {
		spinner, err := pterm.DefaultSpinner.Start("Waiting services availability")
		if err != nil {
			return nil, err
		}

		stack, err = waitStackReady(cmd, apiClient, restoredStackData.GetOrganizationID(), restoredStackData.GetID())
		if err != nil {
			return nil, err
		}

		if err := spinner.Stop(); err != nil {
			return nil, err
		}

		c.store.Stack = stack
	} else {
		c.store.Stack = restoredStackData
	}

	stackClient, err := fctl.NewStackClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
		restoredStackData.GetOrganizationID(),
		restoredStackData.GetID(),
	)
	if err != nil {
		return nil, err
	}

	versions, err := stackClient.GetVersions(cmd.Context())
	if err != nil {
		return nil, err
	}

	if versions.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d when reading versions", versions.StatusCode)
	}

	c.store.Versions = versions.GetVersionsResponse
	c.config = *cfg

	return c, nil
}

func (c *StackRestoreController) Render(cmd *cobra.Command, _ []string) error {
	return internal.PrintStackInformation(cmd.OutOrStdout(), c.store.Stack, c.store.Versions)
}
