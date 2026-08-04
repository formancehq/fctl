package webhooks

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListDeliveryAttemptsStore struct {
	Attempts []DeliveryAttempt `json:"attempts"`
	Cursor   fctl.Cursor       `json:"cursor"`
}

type ListDeliveryAttemptsController struct {
	store *ListDeliveryAttemptsStore
}

func NewListDeliveryAttemptsController() *ListDeliveryAttemptsController {
	return &ListDeliveryAttemptsController{store: &ListDeliveryAttemptsStore{Attempts: []DeliveryAttempt{}}}
}

func (c *ListDeliveryAttemptsController) GetStore() *ListDeliveryAttemptsStore { return c.store }

func (c *ListDeliveryAttemptsController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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
	response, err := api.attempts(cmd.Context(), args[0], cursor, pageSize)
	if err != nil {
		return nil, fmt.Errorf("listing delivery attempts: %w", err)
	}
	c.store.Attempts = response.Cursor.Data
	c.store.Cursor = fctl.Cursor{
		HasMore:  response.Cursor.HasMore,
		PageSize: int64(response.Cursor.PageSize),
		Next:     response.Cursor.Next,
	}
	return c, nil
}

func (c *ListDeliveryAttemptsController) Render(cmd *cobra.Command, _ []string) error {
	table := pterm.TableData{{"ID", "Attempt", "Replay gen.", "Outcome", "Status", "Duration (ms)", "Created at", "Error"}}
	for _, attempt := range c.store.Attempts {
		table = append(table, []string{
			attempt.ID,
			strconv.Itoa(attempt.AttemptNumber),
			strconv.Itoa(attempt.ReplayGeneration),
			attempt.Outcome,
			strconv.Itoa(attempt.StatusCode),
			strconv.FormatInt(attempt.DurationMillis, 10),
			attempt.CreatedAt.Format(time.RFC3339),
			attempt.Error,
		})
	}
	if err := pterm.DefaultTable.WithHasHeader(true).WithWriter(cmd.OutOrStdout()).WithData(table).Render(); err != nil {
		return fmt.Errorf("rendering delivery attempts: %w", err)
	}
	return fctl.RenderCursor(cmd.OutOrStdout(), c.store.Cursor)
}

func NewListDeliveryAttemptsCommand() *cobra.Command {
	return fctl.NewCommand("attempts <delivery-id>",
		fctl.WithShortDescription("List the attempts for a webhook delivery"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithController[*ListDeliveryAttemptsStore](NewListDeliveryAttemptsController()),
	)
}
