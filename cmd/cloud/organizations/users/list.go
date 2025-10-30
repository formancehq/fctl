package users

import (
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	PolicyID int32  `json:"policyID"`
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

	usersResponse, _, err := store.DefaultAPI.ListUsersOfOrganization(cmd.Context(), organizationID).Execute()
	if err != nil {
		return nil, err
	}

	c.store.Users = fctl.Map(usersResponse.Data, func(i membershipclient.OrganizationUser) User {
		return User{
			ID:       i.Id,
			Email:    i.Email,
			PolicyID: i.PolicyID,
		}
	})

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {

	usersRow := fctl.Map(c.store.Users, func(i User) []string {
		return []string{
			i.ID,
			i.Email,
			string(i.PolicyID),
		}
	})

	tableData := fctl.Prepend(usersRow, []string{"ID", "Email", "Role"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
