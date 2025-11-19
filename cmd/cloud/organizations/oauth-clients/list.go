package oauth_clients

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type List struct {
	Cursor components.ReadOrganizationClientsResponseData `json:"cursor"`
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}

	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}

	request := operations.OrganizationClientsReadRequest{
		OrganizationID: organizationID,
	}

	if pageSize > 0 {
		pageSizeInt64 := int64(pageSize)
		request.PageSize = &pageSizeInt64
	}

	if cursor != "" {
		request.Cursor = &cursor
	}

	response, err := apiClient.OrganizationClientsRead(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ReadOrganizationClientsResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Cursor = response.ReadOrganizationClientsResponse.GetData()

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	return showOrganizationClients(cmd.OutOrStdout(), c.store.Cursor)
}
