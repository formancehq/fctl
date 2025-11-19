package users

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type ListStore struct {
	list []components.StackUserAccessResponseData
}
type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{
		list: []components.StackUserAccessResponseData{},
	}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultListStore(),
	}
}

func NewListCommand() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("l"),
		fctl.WithShortDescription("List Stack Access Role within an organization by stacks"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewListController()),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	request := operations.ListStackUsersAccessesRequest{
		OrganizationID: organizationID,
		StackID:        stackID,
	}

	response, err := apiClient.ListStackUsersAccesses(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.StackUserAccessResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.list = response.StackUserAccessResponse.GetData()

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	stackUserAccessMap := fctl.Map(c.store.list, func(o components.StackUserAccessResponseData) []string {
		return []string{
			o.GetStackID(),
			o.GetUserID(),
			o.GetEmail(),
			fmt.Sprintf("%d", o.GetPolicyID()),
		}
	})

	tableData := fctl.Prepend(stackUserAccessMap, []string{"Stack Id", "User Id", "Email", "Role"})

	return pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(tableData).Render()

}
