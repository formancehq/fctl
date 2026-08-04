package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	deliveryStatusFlag  = "status"
	configIDFlag        = "config-id"
	createdAtFromFlag   = "created-at-from"
	createdAtToFlag     = "created-at-to"
	idempotencyKeyFlag  = "idempotency-key"
	endpointFlag        = "endpoint"
	maxDeliveryPageSize = 1000
)

type DeliveryStatus string

const (
	DeliveryStatusPending    DeliveryStatus = "pending"
	DeliveryStatusDelivering DeliveryStatus = "delivering"
	DeliveryStatusSucceeded  DeliveryStatus = "succeeded"
	DeliveryStatusFailed     DeliveryStatus = "failed"
	DeliveryStatusCancelled  DeliveryStatus = "cancelled"
)

type Delivery struct {
	ID               string         `json:"id"`
	EventID          string         `json:"eventID"`
	IdempotencyKey   string         `json:"idempotencyKey,omitempty"`
	ConfigID         string         `json:"configID"`
	EventType        string         `json:"eventType"`
	Payload          string         `json:"payload,omitempty"`
	Status           DeliveryStatus `json:"status"`
	AttemptCount     int            `json:"attemptCount"`
	ReplayGeneration int            `json:"replayGeneration"`
	CycleStartedAt   *time.Time     `json:"cycleStartedAt,omitempty"`
	NextAttemptAt    *time.Time     `json:"nextAttemptAt,omitempty"`
	ClaimedAt        *time.Time     `json:"claimedAt,omitempty"`
	LastAttemptAt    *time.Time     `json:"lastAttemptAt,omitempty"`
	LastStatusCode   *int           `json:"lastStatusCode,omitempty"`
	LastError        string         `json:"lastError,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type DeliveryAttempt struct {
	ID               string    `json:"id"`
	DeliveryID       string    `json:"deliveryID"`
	AttemptNumber    int       `json:"attemptNumber"`
	ReplayGeneration int       `json:"replayGeneration"`
	Endpoint         string    `json:"endpoint"`
	Outcome          string    `json:"outcome"`
	StatusCode       int       `json:"statusCode"`
	Error            string    `json:"error,omitempty"`
	DurationMillis   int64     `json:"durationMillis,omitempty"`
	ResponseExcerpt  string    `json:"responseExcerpt,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type deliveryCursor[T any] struct {
	PageSize int     `json:"pageSize"`
	HasMore  bool    `json:"hasMore"`
	Next     *string `json:"next,omitempty"`
	Data     []T     `json:"data"`
}

type deliveriesResponse struct {
	Cursor deliveryCursor[Delivery] `json:"cursor"`
}

type deliveryResponse struct {
	Data Delivery `json:"data"`
}

type deliveryAttemptsResponse struct {
	Cursor deliveryCursor[DeliveryAttempt] `json:"cursor"`
}

type replayDeliveriesRequest struct {
	CreatedAtFrom time.Time        `json:"createdAtFrom"`
	CreatedAtTo   *time.Time       `json:"createdAtTo,omitempty"`
	Statuses      []DeliveryStatus `json:"statuses,omitempty"`
	ConfigIDs     []string         `json:"configIds,omitempty"`
	Cursor        *string          `json:"cursor,omitempty"`
	PageSize      int              `json:"pageSize,omitempty"`
}

type ReplayDeliveriesResult struct {
	Replayed    int       `json:"replayed"`
	Expedited   int       `json:"expedited"`
	Skipped     int       `json:"skipped"`
	HasMore     bool      `json:"hasMore"`
	NextCursor  *string   `json:"nextCursor,omitempty"`
	CreatedAtTo time.Time `json:"createdAtTo"`
}

type replayDeliveriesResponse struct {
	Data ReplayDeliveriesResult `json:"data"`
}

type listDeliveriesParams struct {
	ConfigID      string
	Status        DeliveryStatus
	CreatedAtFrom *time.Time
	CreatedAtTo   *time.Time
	Cursor        string
	PageSize      int
}

type deliveriesAPI struct {
	baseURL    string
	httpClient *http.Client
}

func newDeliveriesAPI(baseURL string, httpClient *http.Client) *deliveriesAPI {
	return &deliveriesAPI{baseURL: baseURL, httpClient: httpClient}
}

func (c *deliveriesAPI) list(ctx context.Context, params listDeliveriesParams) (*deliveriesResponse, error) {
	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(params.PageSize))
	if params.ConfigID != "" {
		query.Set("configId", params.ConfigID)
	}
	if params.Status != "" {
		query.Set("status", string(params.Status))
	}
	if params.CreatedAtFrom != nil {
		query.Set("createdAtFrom", params.CreatedAtFrom.Format(time.RFC3339))
	}
	if params.CreatedAtTo != nil {
		query.Set("createdAtTo", params.CreatedAtTo.Format(time.RFC3339))
	}
	if params.Cursor != "" {
		query.Set("cursor", params.Cursor)
	}

	response := &deliveriesResponse{}
	if err := c.do(ctx, http.MethodGet, "/api/webhooks/deliveries", query, "", nil, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *deliveriesAPI) get(ctx context.Context, id string) (*deliveryResponse, error) {
	response := &deliveryResponse{}
	if err := c.do(ctx, http.MethodGet, "/api/webhooks/deliveries/"+url.PathEscape(id), nil, "", nil, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *deliveriesAPI) attempts(ctx context.Context, id, cursor string, pageSize int) (*deliveryAttemptsResponse, error) {
	query := url.Values{"pageSize": []string{strconv.Itoa(pageSize)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	response := &deliveryAttemptsResponse{}
	if err := c.do(ctx, http.MethodGet, "/api/webhooks/deliveries/"+url.PathEscape(id)+"/attempts", query, "", nil, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *deliveriesAPI) replay(ctx context.Context, id, idempotencyKey string) (*deliveryResponse, error) {
	response := &deliveryResponse{}
	if err := c.do(ctx, http.MethodPost, "/api/webhooks/deliveries/"+url.PathEscape(id)+"/replay", nil, idempotencyKey, nil, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *deliveriesAPI) replayMany(ctx context.Context, idempotencyKey string, body replayDeliveriesRequest) (*replayDeliveriesResponse, error) {
	response := &replayDeliveriesResponse{}
	if err := c.do(ctx, http.MethodPost, "/api/webhooks/deliveries/replay", nil, idempotencyKey, body, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *deliveriesAPI) do(ctx context.Context, method, path string, query url.Values, idempotencyKey string, body, result any) error {
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("building webhooks URL: %w", err)
	}
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling webhooks: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		data, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
		if readErr != nil {
			return fmt.Errorf("webhooks returned status %d (reading response: %w)", response.StatusCode, readErr)
		}
		message := strings.TrimSpace(string(data))
		if message == "" {
			return fmt.Errorf("webhooks returned status %d", response.StatusCode)
		}
		return fmt.Errorf("webhooks returned status %d: %s", response.StatusCode, message)
	}

	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding webhooks response: %w", err)
	}
	return nil
}

func authenticatedDeliveriesAPI(cmd *cobra.Command) (*deliveriesAPI, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return newDeliveriesAPI(clients.URI, clients.HTTPClient), nil
}

func NewDeliveriesCommand() *cobra.Command {
	return fctl.NewCommand("deliveries",
		fctl.WithAliases("delivery", "del"),
		fctl.WithShortDescription("Inspect and replay webhook deliveries"),
		fctl.WithChildCommands(
			NewListDeliveriesCommand(),
			NewShowDeliveryCommand(),
			NewListDeliveryAttemptsCommand(),
			NewReplayDeliveryCommand(),
			NewReplayDeliveriesCommand(),
		),
	)
}

func deliveryStatus(value string, replayOnly bool) (DeliveryStatus, error) {
	status := DeliveryStatus(value)
	switch status {
	case "":
		return "", nil
	case DeliveryStatusFailed, DeliveryStatusPending:
		return status, nil
	case DeliveryStatusDelivering, DeliveryStatusSucceeded, DeliveryStatusCancelled:
		if !replayOnly {
			return status, nil
		}
	}
	if replayOnly {
		return "", fmt.Errorf("invalid delivery status %q: expected failed or pending", value)
	}
	return "", fmt.Errorf("invalid delivery status %q", value)
}

func deliveryPageSize(cmd *cobra.Command) (int, error) {
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return 0, err
	}
	if pageSize > maxDeliveryPageSize {
		return 0, fmt.Errorf("page size must not exceed %d", maxDeliveryPageSize)
	}
	return int(pageSize), nil
}

func optionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func optionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func renderDelivery(cmd *cobra.Command, delivery Delivery) error {
	return pterm.DefaultTable.WithWriter(cmd.OutOrStdout()).WithData(pterm.TableData{
		{"ID", delivery.ID},
		{"Event ID", delivery.EventID},
		{"Idempotency key", delivery.IdempotencyKey},
		{"Config ID", delivery.ConfigID},
		{"Event type", delivery.EventType},
		{"Status", string(delivery.Status)},
		{"Attempt count", strconv.Itoa(delivery.AttemptCount)},
		{"Replay generation", strconv.Itoa(delivery.ReplayGeneration)},
		{"Cycle started at", optionalTime(delivery.CycleStartedAt)},
		{"Next attempt at", optionalTime(delivery.NextAttemptAt)},
		{"Claimed at", optionalTime(delivery.ClaimedAt)},
		{"Last attempt at", optionalTime(delivery.LastAttemptAt)},
		{"Last status code", optionalInt(delivery.LastStatusCode)},
		{"Last error", delivery.LastError},
		{"Created at", delivery.CreatedAt.Format(time.RFC3339)},
		{"Updated at", delivery.UpdatedAt.Format(time.RFC3339)},
		{"Payload", delivery.Payload},
	}).Render()
}
