package organizations

import (
	"github.com/formancehq/fctl/cmd/cloud/organizations/internal"
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/pointer"
	"github.com/spf13/cobra"
)

type CreateStore struct {
	Organization *membershipclient.OrganizationExpanded `json:"organization"`
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

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	store, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to create a new organization") {
		return nil, fctl.ErrMissingApproval
	}

	defaultPolicyID := fctl.GetInt(cmd, "default-policy-id")
	domain := fctl.GetString(cmd, "domain")

	orgData := membershipclient.CreateOrganizationRequest{
		Name: args[0],
	}

	if defaultPolicyID != 0 {
		orgData.DefaultPolicyID = pointer.For(int32(defaultPolicyID))
	}

	if domain != "" {
		orgData.Domain = pointer.For(domain)
	}

	response, _, err := store.DefaultAPI.
		CreateOrganization(cmd.Context()).
		CreateOrganizationRequest(orgData).
		Execute()
	if err != nil {
		return nil, err
	}

	c.store.Organization = response.Data

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintOrganization(c.store.Organization)
}
