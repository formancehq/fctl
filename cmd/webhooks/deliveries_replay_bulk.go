package webhooks

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ReplayDeliveriesStore struct {
	Result ReplayDeliveriesResult `json:"result"`
}

type ReplayDeliveriesController struct {
	store *ReplayDeliveriesStore
}

func NewReplayDeliveriesController() *ReplayDeliveriesController {
	return &ReplayDeliveriesController{store: &ReplayDeliveriesStore{}}
}

func (c *ReplayDeliveriesController) GetStore() *ReplayDeliveriesStore { return c.store }

func (c *ReplayDeliveriesController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	if !fctl.CheckStackApprobation(cmd, "You are about to replay a page of webhook deliveries") {
		return nil, fctl.ErrMissingApproval
	}
	createdAtFrom, err := fctl.GetDateTime(cmd, createdAtFromFlag)
	if err != nil {
		return nil, fmt.Errorf("parsing --%s: %w", createdAtFromFlag, err)
	}
	if createdAtFrom == nil {
		return nil, fmt.Errorf("--%s is required", createdAtFromFlag)
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

	statusValues := fctl.GetStringSlice(cmd, deliveryStatusFlag)
	statuses := make([]DeliveryStatus, 0, len(statusValues))
	for _, value := range statusValues {
		status, err := deliveryStatus(strings.ToLower(value), true)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}

	request := replayDeliveriesRequest{
		CreatedAtFrom: *createdAtFrom,
		CreatedAtTo:   createdAtTo,
		Statuses:      statuses,
		ConfigIDs:     fctl.GetStringSlice(cmd, configIDFlag),
		PageSize:      pageSize,
	}
	if cursor != "" {
		request.Cursor = &cursor
	}

	api, err := authenticatedDeliveriesAPI(cmd)
	if err != nil {
		return nil, err
	}
	response, err := api.replayMany(cmd.Context(), fctl.GetString(cmd, idempotencyKeyFlag), request)
	if err != nil {
		return nil, fmt.Errorf("replaying deliveries: %w", err)
	}
	c.store.Result = response.Data
	return c, nil
}

func (c *ReplayDeliveriesController) Render(cmd *cobra.Command, _ []string) error {
	result := c.store.Result
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Delivery replay page processed")
	return pterm.DefaultTable.WithWriter(cmd.OutOrStdout()).WithData(pterm.TableData{
		{"Replayed", strconv.Itoa(result.Replayed)},
		{"Expedited", strconv.Itoa(result.Expedited)},
		{"Skipped", strconv.Itoa(result.Skipped)},
		{"Has more", strconv.FormatBool(result.HasMore)},
		{"Next cursor", func() string {
			if result.NextCursor == nil {
				return ""
			}
			return *result.NextCursor
		}()},
		{"Created at to", result.CreatedAtTo.Format(time.RFC3339)},
	}).Render()
}

func NewReplayDeliveriesCommand() *cobra.Command {
	cmd := fctl.NewCommand("replay-bulk",
		fctl.WithShortDescription("Replay a bounded page of failed or pending webhook deliveries"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(createdAtFromFlag, "", "Replay deliveries created at or after this RFC3339 timestamp"),
		fctl.WithStringFlag(createdAtToFlag, "", "Replay deliveries created at or before this RFC3339 timestamp"),
		fctl.WithStringSliceFlag(deliveryStatusFlag, []string{string(DeliveryStatusFailed), string(DeliveryStatusPending)}, "Delivery statuses to replay (failed, pending)"),
		fctl.WithStringSliceFlag(configIDFlag, []string{}, "Restrict replay to webhook config IDs"),
		fctl.WithStringFlag(idempotencyKeyFlag, "", "Idempotency key for this replay request"),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithConfirmFlag(),
		fctl.WithController[*ReplayDeliveriesStore](NewReplayDeliveriesController()),
	)
	for _, flag := range []string{createdAtFromFlag, idempotencyKeyFlag} {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			panic(err)
		}
	}
	return cmd
}
