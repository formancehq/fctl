package policies

import (
	"fmt"
	"strconv"

	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type DeleteStore struct {
	Success bool `json:"success"`
}

type DeleteController struct {
	store *DeleteStore
}

var _ fctl.Controller[*DeleteStore] = (*DeleteController)(nil)

func NewDefaultDeleteStore() *DeleteStore {
	return &DeleteStore{}
}

func NewDeleteController() *DeleteController {
	return &DeleteController{
		store: NewDefaultDeleteStore(),
	}
}

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand(`delete <policy-id>`,
		fctl.WithAliases("del", "d"),
		fctl.WithShortDescription("Delete a policy"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewDeleteController()),
	)
}

func (c *DeleteController) GetStore() *DeleteStore {
	return c.store
}

func (c *DeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	policyID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid policy ID: %w", err)
	}

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to delete a policy") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.DeletePolicyRequest{
		OrganizationID: organizationID,
		PolicyID:       policyID,
	}

	_, err = apiClient.DeletePolicy(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.Success = true

	return c, nil
}

func (c *DeleteController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Policy deleted successfully")
	return nil
}
