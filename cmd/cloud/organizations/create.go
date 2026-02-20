package organizations

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/v3/cmd/cloud/organizations/internal"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateStore struct {
	Organization *components.OrganizationExpanded `json:"organization"`
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
	return fctl.NewCommand(`create <name> --default-stack-role "ADMIN" --default-organization-role "ADMIN"`,
		fctl.WithAliases("cr", "c"),
		fctl.WithShortDescription("Create organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithIntFlag("default-policy-id", 0, "Default policy id"),
		fctl.WithStringFlag("domain", "", "Organization Domain"),
		fctl.WithConfirmFlag(),
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

	apiClient, err := fctl.NewMembershipClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to create a new organization") {
		return nil, fctl.ErrMissingApproval
	}

	defaultPolicyID := fctl.GetInt(cmd, "default-policy-id")
	domain := fctl.GetString(cmd, "domain")

	orgData := components.CreateOrganizationRequest{
		Name: args[0],
	}

	if defaultPolicyID != 0 {
		orgData.DefaultPolicyID = pointer.For(int64(defaultPolicyID))
	}

	if domain != "" {
		orgData.Domain = pointer.For(domain)
	}

	response, err := apiClient.CreateOrganization(cmd.Context(), &orgData)
	if err != nil {
		return nil, err
	}

	if response.CreateOrganizationResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Organization = response.CreateOrganizationResponse.GetData()

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintOrganization(c.store.Organization)
}
