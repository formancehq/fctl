package schedules

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Cursor *payments.V3ConnectorSchedulesCursorResponseCursor `json:"cursor"`
}

type ListController struct {
	PaymentsVersion versions.Version

	store *ListStore
}

func (c *ListController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListStore() *ListStore {
	return &ListStore{
		Cursor: &payments.V3ConnectorSchedulesCursorResponseCursor{},
	}
}

func NewListController() *ListController {
	return &ListController{
		store: NewListStore(),
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list <connectorID>",
		fctl.WithAliases("ls", "l"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("List connector schedules"),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithController[*ListStore](c),
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

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion.Major < versions.V3 {
		return nil, fmt.Errorf("connector schedules are only supported in >= v3.0.0")
	}

	connectorID := args[0]
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}

	request := operations.V3ListConnectorSchedulesRequest{
		ConnectorID: connectorID,
		PageSize:    fctl.Ptr(int64(pageSize)),
	}
	if cursor != "" {
		request.Cursor = &cursor
	}

	response, err := stackClient.Payments.V3.ListConnectorSchedules(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	if response.V3ConnectorSchedulesCursorResponse == nil {
		return nil, fmt.Errorf("unexpected response: %v", response)
	}

	c.store.Cursor = &response.V3ConnectorSchedulesCursorResponse.Cursor

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	tableData := fctl.Map(c.store.Cursor.Data, func(s payments.V3Schedule) []string {
		return []string{
			s.ID,
			s.CreatedAt.Format(time.RFC3339),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "CreatedAt"})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return fctl.RenderCursor(cmd.OutOrStdout(), fctl.Cursor{
		HasMore:  c.store.Cursor.HasMore,
		PageSize: c.store.Cursor.PageSize,
		Next:     c.store.Cursor.Next,
		Previous: c.store.Cursor.Previous,
	})
}
