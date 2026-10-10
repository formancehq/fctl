package modules

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func newWebhooks(client *http.Client) pluginsdk.Plugin {
	listOp := leaf("list", "list", http.MethodGet, "configs", 0, 0, str("config-id", "Config ID filter"), str("endpoint", "Endpoint URL filter"))
	listOp.query = map[string]string{"config-id": "id", "endpoint": "endpoint"}
	create := confirmed(leaf("create", "create <endpoint> <event-type>...", http.MethodPost, "configs", 2, 1000, str("secret", "Optional base64 signing secret (24 bytes)")))
	create.body = webhookConfig
	update := confirmed(leaf("update", "update <config-id> <endpoint> <event-type>...", http.MethodPut, "configs/$0", 3, 1001, str("secret", "Optional signing secret")))
	update.body = webhookConfig
	secret := confirmed(leaf("change-secret", "change-secret <config-id> [secret]", http.MethodPut, "configs/$0/secret/change", 1, 2))
	secret.body = func(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
		value := ""
		if len(r.Args) == 2 {
			value = r.Args[1]
		}
		if err := signingSecret(value); err != nil {
			return nil, err
		}
		return marshal(map[string]string{"secret": value})
	}
	operations := []operation{listOp, create, update, secret,
		confirmed(leaf("activate", "activate <config-id>", http.MethodPut, "configs/$0/activate", 1, 1)),
		confirmed(leaf("deactivate", "deactivate <config-id>", http.MethodPut, "configs/$0/deactivate", 1, 1)),
		confirmed(leaf("delete", "delete <config-id>", http.MethodDelete, "configs/$0", 1, 1)),
		confirmed(leaf("test", "test <config-id>", http.MethodGet, "configs/$0/test", 1, 1)),
		leaf("deliveries show", "show <delivery-id>", http.MethodGet, "deliveries/$0", 1, 1),
	}
	deliveries := leaf("deliveries list", "list", http.MethodGet, "deliveries", 0, 0, append(pagination(),
		str("config-id", "Config ID filter"), str("status", "pending, delivering, succeeded, failed or cancelled"),
		str("created-at-from", "Inclusive RFC3339 start"), str("created-at-to", "Inclusive RFC3339 end"))...)
	deliveries.query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "config-id": "configId", "status": "status", "created-at-from": "createdAtFrom", "created-at-to": "createdAtTo"}
	deliveries.body = func(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
		if r.Body != nil {
			return nil, fmt.Errorf("list does not accept a body")
		}
		if v := r.Flags["status"]; v != "" && !slices.Contains([]string{"pending", "delivering", "succeeded", "failed", "cancelled"}, v) {
			return nil, fmt.Errorf("invalid delivery status %q", v)
		}
		return nil, timeRange(r.Flags["created-at-from"], r.Flags["created-at-to"])
	}
	attempts := leaf("deliveries attempts", "attempts <delivery-id>", http.MethodGet, "deliveries/$0/attempts", 1, 1, pagination()...)
	attempts.query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
	keyFlag := str("idempotency-key", "Required replay idempotency key")
	keyFlag.Required = true
	replay := confirmed(leaf("deliveries replay", "replay <delivery-id>", http.MethodPost, "deliveries/$0/replay", 1, 1, keyFlag))
	bulk := confirmed(leaf("deliveries replay-bulk", "replay-bulk", http.MethodPost, "deliveries/replay", 0, 0, append(pagination(), keyFlag,
		str("created-at-from", "Required inclusive RFC3339 start"), str("created-at-to", "Inclusive RFC3339 end"),
		str("status", "CSV or JSON array; failed and pending by default"), str("config-id", "CSV or JSON array of config IDs"))...))
	bulk.body = replayDeliveriesBody
	bulk.spec.Inputs = []pluginsdk.InputSpec{{Title: "Replay from (RFC3339)", Kind: "input", Flag: "created-at-from", Required: true}, {Title: "Idempotency key", Kind: "input", Flag: "idempotency-key", Required: true}}
	operations = append(operations, deliveries, attempts, replay, bulk)
	return newPlugin("webhooks", client, operations)
}

func webhookConfig(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil {
		return nil, fmt.Errorf("use endpoint and event-type arguments")
	}
	offset := 0
	if r.CommandPath[len(r.CommandPath)-1] == "update" {
		offset = 1
	}
	endpoint := r.Args[offset]
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return nil, fmt.Errorf("webhook endpoint must be an absolute HTTP(S) URL")
	}
	for _, event := range r.Args[offset+1:] {
		if strings.TrimSpace(event) == "" {
			return nil, fmt.Errorf("event types cannot be empty")
		}
	}
	body := map[string]any{"endpoint": endpoint, "eventTypes": r.Args[offset+1:]}
	secret := r.Flags["secret"]
	if err := signingSecret(secret); err != nil {
		return nil, err
	}
	if offset == 0 || secret != "" || r.ChangedFlags["secret"] {
		body["secret"] = secret
	}
	return marshal(body)
}

func signingSecret(value string) error {
	if value == "" {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != 24 {
		return fmt.Errorf("signing secret must be base64 encoding of 24 bytes")
	}
	return nil
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
	statuses, err := list(r.Flags["status"])
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
	ids, err := list(r.Flags["config-id"])
	if err != nil {
		return nil, err
	}
	page := json.Number(r.Flags["page-size"])
	if _, err := queryValues(map[string]string{"page-size": "pageSize"}, r); err != nil {
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
	return marshal(body)
}
