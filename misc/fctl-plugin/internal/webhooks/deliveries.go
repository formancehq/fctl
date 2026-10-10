package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func deliveryOperations() []command.Operation {
	deliveries := command.Leaf("deliveries list", "list", http.MethodGet, "deliveries", 0, 0, append(command.PaginationFlags(),
		command.StringFlag("config-id", "Config ID filter"), command.StringFlag("status", "pending, delivering, succeeded, failed or cancelled"),
		command.StringFlag("created-at-from", "Inclusive RFC3339 start"), command.StringFlag("created-at-to", "Inclusive RFC3339 end"))...)
	deliveries.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "config-id": "configId", "status": "status", "created-at-from": "createdAtFrom", "created-at-to": "createdAtTo"}
	deliveries.Body = deliveryListBody
	attempts := command.Leaf("deliveries attempts", "attempts <delivery-id>", http.MethodGet, "deliveries/$0/attempts", 1, 1, command.PaginationFlags()...)
	attempts.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
	keyFlag := command.StringFlag("idempotency-key", "Required replay idempotency key")
	keyFlag.Required = true
	replay := command.Confirmed(command.Leaf("deliveries replay", "replay <delivery-id>", http.MethodPost, "deliveries/$0/replay", 1, 1, keyFlag))
	bulk := command.Confirmed(command.Leaf("deliveries replay-bulk", "replay-bulk", http.MethodPost, "deliveries/replay", 0, 0, append(command.PaginationFlags(), keyFlag,
		command.StringFlag("created-at-from", "Required inclusive RFC3339 start"), command.StringFlag("created-at-to", "Inclusive RFC3339 end"),
		command.StringFlag("status", "CSV or JSON array; failed and pending by default"), command.StringFlag("config-id", "CSV or JSON array of config IDs"))...))
	bulk.Body = replayDeliveriesBody
	bulk.Spec.Inputs = []pluginsdk.InputSpec{{Title: "Replay from (RFC3339)", Kind: "input", Flag: "created-at-from", Required: true}, {Title: "Idempotency key", Kind: "input", Flag: "idempotency-key", Required: true}}
	return []command.Operation{
		command.Leaf("deliveries show", "show <delivery-id>", http.MethodGet, "deliveries/$0", 1, 1),
		deliveries, attempts, replay, bulk,
	}

}

func deliveryListBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil {
		return nil, fmt.Errorf("list does not accept a body")
	}
	if v := r.Flags["status"]; v != "" && !slices.Contains([]string{"pending", "delivering", "succeeded", "failed", "cancelled"}, v) {
		return nil, fmt.Errorf("invalid delivery status %q", v)
	}
	return nil, timeRange(r.Flags["created-at-from"], r.Flags["created-at-to"])
}

func timeRange(from, to string) error {
	var start, end time.Time
	var err error
	if from != "" {
		start, err = time.Parse(time.RFC3339Nano, from)
		if err != nil {
			return fmt.Errorf("start must be RFC3339: %w", err)
		}
	}
	if to != "" {
		end, err = time.Parse(time.RFC3339Nano, to)
		if err != nil {
			return fmt.Errorf("end must be RFC3339: %w", err)
		}
	}
	if from != "" && to != "" && end.Before(start) {
		return fmt.Errorf("end timestamp precedes start timestamp")
	}
	return nil
}

func replayDeliveriesBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil {
		return nil, fmt.Errorf("replay-bulk uses its declared flags")
	}
	from, to := r.Flags["created-at-from"], r.Flags["created-at-to"]
	if from == "" {
		return nil, fmt.Errorf("--created-at-from is required")
	}
	if err := timeRange(from, to); err != nil {
		return nil, err
	}
	statuses, err := command.List(r.Flags["status"])
	if err != nil {
		return nil, err
	}
	if len(statuses) == 0 {
		statuses = []string{"failed", "pending"}
	}
	for i, status := range statuses {
		statuses[i] = strings.ToLower(status)
		if statuses[i] != "failed" && statuses[i] != "pending" {
			return nil, fmt.Errorf("replay status must be failed or pending")
		}
	}
	ids, err := command.List(r.Flags["config-id"])
	if err != nil {
		return nil, err
	}
	page := json.Number(r.Flags["page-size"])
	if _, err := command.QueryValues(map[string]string{"page-size": "pageSize"}, r); err != nil {
		return nil, err
	}
	body := map[string]any{"createdAtFrom": from, "statuses": statuses, "pageSize": page}
	if to != "" {
		body["createdAtTo"] = to
	}
	if len(ids) > 0 {
		body["configIds"] = ids
	}
	if r.Flags["cursor"] != "" {
		body["cursor"] = r.Flags["cursor"]
	}
	return json.Marshal(body)
}
