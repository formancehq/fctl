package policies

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type CreateStore struct {
	Policy *components.Policy `json:"policy"`
}

type CreateController struct {
	store *CreateStore
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand(`create <name>`,
		fctl.WithAliases("c", "cr"),
		fctl.WithShortDescription("Create a policy"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("description", "", "Policy description"),
		fctl.WithConfirmFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewCreateController()),
	)
}

func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to create a new policy") {
		return nil, fctl.ErrMissingApproval
	}

	description := fctl.GetString(cmd, "description")
	policyData := components.PolicyData{
		Name: args[0],
	}
	if description != "" {
		policyData.Description = pointer.For(description)
	}

	request := operations.CreatePolicyRequest{
		OrganizationID: organizationID,
		Body:           &policyData,
	}

	response, err := apiClient.CreatePolicy(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.CreatePolicyResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Policy = response.CreatePolicyResponse.GetData()

	return c, nil
}

func (c *CreateController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Policy created successfully")
	return nil
}
