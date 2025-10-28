package policies

import (
	"fmt"
	"strconv"

	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type AddScopeStore struct {
	Success bool `json:"success"`
}

type AddScopeController struct {
	store *AddScopeStore
}

var _ fctl.Controller[*AddScopeStore] = (*AddScopeController)(nil)

func NewDefaultAddScopeStore() *AddScopeStore {
	return &AddScopeStore{}
}

func NewAddScopeController() *AddScopeController {
	return &AddScopeController{
		store: NewDefaultAddScopeStore(),
	}
}

func NewAddScopeCommand() *cobra.Command {
	return fctl.NewCommand(`add-scope <policy-id> <scope-id>`,
		fctl.WithAliases("add", "a"),
		fctl.WithShortDescription("Add a scope to a policy"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithConfirmFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewAddScopeController()),
	)
}

func (c *AddScopeController) GetStore() *AddScopeStore {
	return c.store
}

func (c *AddScopeController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to add a scope to a policy") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.AddScopeToPolicyRequest{
		OrganizationID: organizationID,
		PolicyID:       policyID,
		ScopeID:        scopeID,
	}

	_, err = apiClient.AddScopeToPolicy(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.Success = true

	return c, nil
}

func (c *AddScopeController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Scope added to policy successfully")
	return nil
}
