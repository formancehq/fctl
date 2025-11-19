package organizations

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/cmd/cloud/organizations/internal"
	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type DescribeStore struct {
	*components.OrganizationExpanded
}
type DescribeController struct {
	store *DescribeStore
}

var _ fctl.Controller[*DescribeStore] = (*DescribeController)(nil)

func NewDefaultDescribeStore() *DescribeStore {
	return &DescribeStore{}
}

func NewDescribeController() *DescribeController {
	return &DescribeController{
		store: NewDefaultDescribeStore(),
	}
}

func NewDescribeCommand() *cobra.Command {
	return fctl.NewCommand("describe <organizationId>",
		fctl.WithShortDescription("Describe organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithBoolFlag("expand", false, "Expand the organization"),
		fctl.WithValidArgsFunction(fctl.OrganizationCompletion),
		fctl.WithController(NewDescribeController()),
	)
}

func (c *DescribeController) GetStore() *DescribeStore {
	return c.store
}

func (c *DescribeController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, args[0])
	if err != nil {
		return nil, err
	}

	expand := fctl.GetBool(cmd, "expand")
	request := operations.ReadOrganizationRequest{
		OrganizationID: args[0],
		Expand:         pointer.For(expand),
	}

	response, err := apiClient.ReadOrganization(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ReadOrganizationResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.OrganizationExpanded = response.ReadOrganizationResponse.GetData()
	return c, nil
}

func (c *DescribeController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintOrganization(c.store.OrganizationExpanded)
}
