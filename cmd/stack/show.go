package stack

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/stack/internal"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

var errStackNotFound = errors.New("stack not found")

type StackShowStore struct {
	Stack    *components.Stack           `json:"stack"`
	Versions *shared.GetVersionsResponse `json:"versions"`
}

type StackShowController struct {
	store  *StackShowStore
	config fctl.Config
}

var _ fctl.Controller[*StackShowStore] = (*StackShowController)(nil)

func NewDefaultStackShowStore() *StackShowStore {
	return &StackShowStore{
		Stack: &components.Stack{},
	}
}

func NewStackShowController() *StackShowController {
	return &StackShowController{
		store: NewDefaultStackShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	var stackNameFlag = "name"

	return fctl.NewMembershipCommand("show (<stack-id> | --name=<stack-name>)",
		fctl.WithAliases("s", "sh"),
		fctl.WithShortDescription("Show stack"),
		fctl.WithArgs(cobra.MaximumNArgs(1)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithStringFlag(stackNameFlag, "", ""),
		fctl.WithController(NewStackShowController()),
	)
}

func (c *StackShowController) GetStore() *StackShowStore {
	return c.store
}

func (c *StackShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	var stackNameFlag = "name"
	var stack *components.Stack

	cfg, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if len(args) == 1 {
		if fctl.GetString(cmd, stackNameFlag) != "" {
			return nil, errors.New("need either an id of a name specified using --name flag")
		}
		getRequest := operations.GetStackRequest{
			OrganizationID: organizationID,
			StackID:        args[0],
		}
		stackResponse, err := apiClient.GetStack(cmd.Context(), getRequest)
		if err != nil {
			if stackResponse.GetHTTPMeta().Response.StatusCode == http.StatusNotFound {
				return nil, errStackNotFound
			}
			return nil, fmt.Errorf("listing stacks: %w", err)
		}
		if stackResponse.ReadStackResponse == nil {
			return nil, fmt.Errorf("unexpected response: no data")
		}
		stack = stackResponse.ReadStackResponse.GetData()
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
		return nil, errStackNotFound
	}

	c.store.Stack = stack
	c.config = *cfg

	// the stack is not active, we can't get the running versions
	// Maybe add something in the process with sync status and store it in membership
	if stack.GetStatus() != "ACTIVE" {
		return c, nil
	}

	stackClient, err := fctl.NewStackClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
		stack.GetOrganizationID(),
		stack.GetID(),
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

	return c, nil

}

func (c *StackShowController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintStackInformation(cmd.OutOrStdout(), c.store.Stack, c.store.Versions)
}
