package organizations

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/v3/cmd/cloud/organizations/internal"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UpdateController struct {
	store *DescribeStore
}

var _ fctl.Controller[*DescribeStore] = (*UpdateController)(nil)

func NewDefaultUpdateStore() *DescribeStore {
	return &DescribeStore{}
}

func NewUpdateController() *UpdateController {
	return &UpdateController{
		store: NewDefaultUpdateStore(),
	}
}

func NewUpdateCommand() *cobra.Command {
	return fctl.NewCommand("update <organizationId> --name <name> --default-policy-id <defaultPolicyID...>",
		fctl.WithAliases("update"),
		fctl.WithShortDescription("Update organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(fctl.OrganizationCompletion),
		fctl.WithConfirmFlag(),
		fctl.WithStringFlag("name", "", "Organization Name"),
		fctl.WithIntFlag("default-policy-id", 0, "Default policy id"),
		fctl.WithStringFlag("domain", "", "Organization Domain"),
		fctl.WithController(NewUpdateController()),
	)
}

func (c *UpdateController) GetStore() *DescribeStore {
	return c.store
}

func (c *UpdateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, args[0])
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to update an organization") {
		return nil, fctl.ErrMissingApproval
	}

	readRequest := operations.ReadOrganizationRequest{
		OrganizationID: args[0],
	}

	org, err := apiClient.ReadOrganization(cmd.Context(), readRequest)
	if err != nil {
		return nil, err
	}

	if org.ReadOrganizationResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	orgData := org.ReadOrganizationResponse.GetData()

	preparedData := components.OrganizationData{
		Name: func() string {
			if cmd.Flags().Changed("name") {
				return cmd.Flag("name").Value.String()
			}
			return orgData.GetName()
		}(),
		DefaultPolicyID: func() *int64 {
			if cmd.Flags().Changed("default-policy-id") {
				return pointer.For(int64(fctl.GetInt(cmd, "default-policy-id")))
			}
			return orgData.GetDefaultPolicyID()
		}(),
		Domain: func() *string {
			str := fctl.GetString(cmd, "domain")
			if str != "" {
				return &str
			}
			return orgData.GetDomain()
		}(),
	}

	updateRequest := operations.UpdateOrganizationRequest{
		OrganizationID: args[0],
		Body:           &preparedData,
	}

	response, err := apiClient.UpdateOrganization(cmd.Context(), updateRequest)
	if err != nil {
		return nil, err
	}

	if response.ReadOrganizationResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.OrganizationExpanded = response.ReadOrganizationResponse.GetData()

	return c, nil
}

func (c *UpdateController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintOrganization(c.store.OrganizationExpanded)
}
