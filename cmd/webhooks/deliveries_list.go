package webhooks

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListDeliveriesStore struct {
	Deliveries []Delivery  `json:"deliveries"`
	Cursor     fctl.Cursor `json:"cursor"`
}

type ListDeliveriesController struct {
	store *ListDeliveriesStore
}

func NewListDeliveriesController() *ListDeliveriesController {
	return &ListDeliveriesController{store: &ListDeliveriesStore{Deliveries: []Delivery{}}}
}

func (c *ListDeliveriesController) GetStore() *ListDeliveriesStore { return c.store }

func (c *ListDeliveriesController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	status, err := deliveryStatus(fctl.GetString(cmd, deliveryStatusFlag), false)
	if err != nil {
		return nil, err
	}
	createdAtFrom, err := fctl.GetDateTime(cmd, createdAtFromFlag)
	if err != nil {
		return nil, fmt.Errorf("parsing --%s: %w", createdAtFromFlag, err)
	}
	createdAtTo, err := fctl.GetDateTime(cmd, createdAtToFlag)
	if err != nil {
		return nil, fmt.Errorf("parsing --%s: %w", createdAtToFlag, err)
	}
	pageSize, err := deliveryPageSize(cmd)
	if err != nil {
		return nil, err
	}
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}

	api, err := authenticatedDeliveriesAPI(cmd)
	if err != nil {
		return nil, err
	}
	response, err := api.list(cmd.Context(), listDeliveriesParams{
		ConfigID:      fctl.GetString(cmd, configIDFlag),
		Status:        status,
		CreatedAtFrom: createdAtFrom,
		CreatedAtTo:   createdAtTo,
		Cursor:        cursor,
		PageSize:      pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("listing deliveries: %w", err)
	}

	c.store.Deliveries = response.Cursor.Data
	c.store.Cursor = fctl.Cursor{
		HasMore:  response.Cursor.HasMore,
		PageSize: int64(response.Cursor.PageSize),
		Next:     response.Cursor.Next,
	}
	return c, nil
}

func (c *ListDeliveriesController) Render(cmd *cobra.Command, _ []string) error {
	table := pterm.TableData{{"ID", "Config ID", "Event type", "Status", "Attempts", "Replay gen.", "Last status", "Created at"}}
	for _, delivery := range c.store.Deliveries {
		table = append(table, []string{
			delivery.ID,
			delivery.ConfigID,
			delivery.EventType,
			string(delivery.Status),
			strconv.Itoa(delivery.AttemptCount),
			strconv.Itoa(delivery.ReplayGeneration),
			optionalInt(delivery.LastStatusCode),
			delivery.CreatedAt.Format(time.RFC3339),
		})
	}
	if err := pterm.DefaultTable.WithHasHeader(true).WithWriter(cmd.OutOrStdout()).WithData(table).Render(); err != nil {
		return fmt.Errorf("rendering deliveries: %w", err)
	}
	return fctl.RenderCursor(cmd.OutOrStdout(), c.store.Cursor)
}

func NewListDeliveriesCommand() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List webhook deliveries"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(configIDFlag, "", "Filter by webhook config ID"),
		fctl.WithStringFlag(deliveryStatusFlag, "", "Filter by status (pending, delivering, succeeded, failed, cancelled)"),
		fctl.WithStringFlag(createdAtFromFlag, "", "Filter deliveries created at or after this RFC3339 timestamp"),
		fctl.WithStringFlag(createdAtToFlag, "", "Filter deliveries created at or before this RFC3339 timestamp"),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithController[*ListDeliveriesStore](NewListDeliveriesController()),
	)
}
