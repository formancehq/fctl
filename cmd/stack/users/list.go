package users

import (
	"fmt"
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	listStackUsersAccesses, response, err := store.DefaultAPI.ListStackUsersAccesses(cmd.Context(), organizationID, stackID).Execute()
	if err != nil {
		return nil, err
	}

	if response.StatusCode > 300 {
		return nil, err
	}

	c.store.list = append(c.store.list, listStackUsersAccesses.Data...)

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	stackUserAccessMap := fctl.Map(c.store.list, func(o membershipclient.StackUserAccess) []string {
		return []string{
			o.StackId,
			o.UserId,
			o.Email,
			fmt.Sprintf("%d", o.PolicyID),
		}
	})

	tableData := fctl.Prepend(stackUserAccessMap, []string{"Stack Id", "User Id", "Email", "Role"})

	return pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(tableData).Render()

}
