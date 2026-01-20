package policies

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type UpdateStore struct {
	Policy *components.Policy `json:"policy"`
}

type UpdateController struct {
	store *UpdateStore
}

var _ fctl.Controller[*UpdateStore] = (*UpdateController)(nil)

func NewDefaultUpdateStore() *UpdateStore {
	return &UpdateStore{}
}

func NewUpdateController() *UpdateController {
	return &UpdateController{
		store: NewDefaultUpdateStore(),
	}
}

func NewUpdateCommand() *cobra.Command {
	return fctl.NewCommand(`update <policy-id>`,
		fctl.WithAliases("u", "up"),
		fctl.WithShortDescription("Update a policy"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("name", "", "Policy name"),
		fctl.WithStringFlag("description", "", "Policy description"),
		fctl.WithConfirmFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewUpdateController()),
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

	policyID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid policy ID: %w", err)
	}

	// Read current policy to get existing values
	readRequest := operations.ReadPolicyRequest{
		OrganizationID: organizationID,
		PolicyID:       policyID,
	}

	readResponse, err := apiClient.ReadPolicy(cmd.Context(), readRequest)
	if err != nil {
		return nil, err
	}

	if readResponse.ReadPolicyResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	currentPolicy := readResponse.ReadPolicyResponse.GetData()

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to update a policy") {
		return nil, fctl.ErrMissingApproval
	}

	// Prepare update data
	name := fctl.GetString(cmd, "name")
	if name == "" {
		name = currentPolicy.GetName()
	}

	description := fctl.GetString(cmd, "description")
	policyData := components.CreatePolicyRequest{
		Name: name,
	}
	if description != "" || cmd.Flags().Changed("description") {
		if description != "" {
			policyData.Description = pointer.For(description)
		} else {
			policyData.Description = currentPolicy.GetDescription()
		}
	} else {
		policyData.Description = currentPolicy.GetDescription()
	}

	updateRequest := operations.UpdatePolicyRequest{
		OrganizationID: organizationID,
		PolicyID:       policyID,
		Body:           &policyData,
	}

	response, err := apiClient.UpdatePolicy(cmd.Context(), updateRequest)
	if err != nil {
		return nil, err
	}

	if response.UpdatePolicyResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Policy = response.UpdatePolicyResponse.GetData()

	return c, nil
}

func (c *UpdateController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Policy updated successfully")
	return nil
}
