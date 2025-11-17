package users

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
)

type ListStore struct {
	list []membershipclient.StackUserAccess
}
type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{
		list: []membershipclient.StackUserAccess{},
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

	store := fctl.GetMembershipStackStore(cmd.Context())

	listStackUsersAccesses, response, err := store.Client().ListStackUsersAccesses(cmd.Context(), store.OrganizationId(), store.StackId()).Execute()
	if err != nil {
		return nil, err
	}

	if response.StatusCode > 300 {
		return nil, err
	}

	c.store.list = append(c.store.list, listStackUsersAccesses.Data...)

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	stackUserAccessMap := fctl.Map(c.store.list, func(o membershipclient.StackUserAccess) []string {
		return []string{
			o.StackId,
			o.UserId,
			o.Email,
			string(o.Role),
		}
	})

	tableData := fctl.Prepend(stackUserAccessMap, []string{"Stack Id", "User Id", "Email", "Role"})

	return pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(tableData).Render()

}
