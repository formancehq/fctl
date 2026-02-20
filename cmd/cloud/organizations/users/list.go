package users

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	PolicyID int64  `json:"policyID"`
}

type ListStore struct {
	Users []User `json:"users"`
}
type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultListStore(),
	}
}

func NewListCommand() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("List users"),
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

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	request := operations.ListUsersOfOrganizationRequest{
		OrganizationID: organizationID,
	}

	response, err := apiClient.ListUsersOfOrganization(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ListUsersResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Users = fctl.Map(response.ListUsersResponse.GetData(), func(i components.OrganizationUser) User {
		return User{
			ID:       i.GetID(),
			Email:    i.GetEmail(),
			PolicyID: i.GetPolicyID(),
		}
	})

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {

	usersRow := fctl.Map(c.store.Users, func(i User) []string {
		return []string{
			i.ID,
			i.Email,
			fmt.Sprint(i.PolicyID),
		}
	})

	tableData := fctl.Prepend(usersRow, []string{"ID", "Email", "Policy ID"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
