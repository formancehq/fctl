package oauth_clients

import (
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

type List struct {
	Cursor membershipclient.ReadOrganizationClientsResponseData `json:"cursor"`
}

type ListController struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListController)(nil)

func NewDefaultList() *List {
	return &List{}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultList(),
	}
}

func NewListCommand() *cobra.Command {
	return fctl.NewCommand(`list`,
		fctl.WithShortDescription("List organization OAuth clients"),
		fctl.WithPageSizeFlag(),
		fctl.WithCursorFlag(),
		fctl.WithController(NewListController()),
	)
}

func (c *ListController) GetStore() *List {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetMembershipStore(cmd.Context())
	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	req := store.Client().OrganizationClientsRead(cmd.Context(), organizationID)

	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}

	if pageSize > 0 {
		req = req.PageSize(pageSize)
	}

	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}

	if cursor != "" {
		req = req.Cursor(cursor)
	}

	response, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	c.store.Cursor = response.Data

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	return showOrganizationClients(cmd.OutOrStdout(), c.store.Cursor)
}
