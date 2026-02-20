package applications

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/time"
	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Applications []components.Application `json:"applications"`
	Cursor       *components.Cursor       `json:"cursor"`
}

type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{
		Applications: []components.Application{},
	}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultListStore(),
	}
}

func NewListCommand() *cobra.Command {
	return fctl.NewCommand(`list`,
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List applications available for organization"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithIntFlag("page", 0, "Page number"),
		fctl.WithIntFlag("page-size", 15, "Page size"),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewListController()),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	page := fctl.GetInt(cmd, "page")
	pageSize := fctl.GetInt(cmd, "page-size")

	request := operations.ListOrganizationApplicationsRequest{
		OrganizationID: organizationID,
		Page:           pointer.For(int64(page)),
		PageSize:       pointer.For(int64(pageSize)),
	}

	response, err := apiClient.ListOrganizationApplications(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ListApplicationsResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	cursor := response.ListApplicationsResponse.GetCursor()
	if cursor == nil {
		return nil, fmt.Errorf("unexpected response: no cursor data")
	}

	c.store.Applications = cursor.GetData()
	c.store.Cursor = cursor

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	if len(c.store.Applications) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No applications found.")
		return nil
	}

	header := []string{"ID", "Name", "Alias", "URL", "Description", "Created At", "Updated At"}
	tableData := fctl.Map(c.store.Applications, func(app components.Application) []string {
		return []string{
			app.GetID(),
			app.GetName(),
			app.GetAlias(),
			app.GetURL(),
			func() string {
				if desc := app.GetDescription(); desc != nil {
					return *desc
				}
				return ""
			}(),
			time.Time{Time: app.GetCreatedAt()}.String(),
			time.Time{Time: app.GetUpdatedAt()}.String(),
		}
	})

	tableData = fctl.Prepend(tableData, header)

	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	// Display cursor information if available
	if c.store.Cursor != nil {
		cursorInfo := [][]string{
			{"Has More", fctl.BoolToString(c.store.Cursor.GetHasMore())},
			{"Page Size", fmt.Sprintf("%d", c.store.Cursor.GetPageSize())},
		}
		if next := c.store.Cursor.GetNext(); next != nil {
			cursorInfo = append(cursorInfo, []string{"Next", *next})
		}
		if previous := c.store.Cursor.GetPrevious(); previous != nil {
			cursorInfo = append(cursorInfo, []string{"Previous", *previous})
		}

		fmt.Fprintln(cmd.OutOrStdout(), "")
		if err := pterm.DefaultTable.
			WithWriter(cmd.OutOrStdout()).
			WithData(cursorInfo).
			Render(); err != nil {
			return err
		}
	}

	return nil
}
