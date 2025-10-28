package policies

import (
	"fmt"
	"strconv"

	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type RemoveScopeStore struct {
	Success bool `json:"success"`
}

type RemoveScopeController struct {
	store *RemoveScopeStore
}

var _ fctl.Controller[*RemoveScopeStore] = (*RemoveScopeController)(nil)

func NewDefaultRemoveScopeStore() *RemoveScopeStore {
	return &RemoveScopeStore{}
}

func NewRemoveScopeController() *RemoveScopeController {
	return &RemoveScopeController{
		store: NewDefaultRemoveScopeStore(),
	}
}

func NewRemoveScopeCommand() *cobra.Command {
	return fctl.NewCommand(`remove-scope <policy-id> <scope-id>`,
		fctl.WithAliases("remove", "rm", "r"),
		fctl.WithShortDescription("Remove a scope from a policy"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithConfirmFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewRemoveScopeController()),
	)
}

func (c *RemoveScopeController) GetStore() *RemoveScopeStore {
	return c.store
}

func (c *RemoveScopeController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	policyID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid policy ID: %w", err)
	}

	scopeID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid scope ID: %w", err)
	}

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to remove a scope from a policy") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.RemoveScopeFromPolicyRequest{
		OrganizationID: organizationID,
		PolicyID:       policyID,
		ScopeID:        scopeID,
	}

	_, err = apiClient.RemoveScopeFromPolicy(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.Success = true

	return c, nil
}

func (c *RemoveScopeController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Scope removed from policy successfully")
	return nil
}
