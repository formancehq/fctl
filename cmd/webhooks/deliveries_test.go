package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestDeliveriesAPIReadEndpoints(t *testing.T) {
	from := time.Date(2026, time.July, 1, 10, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/webhooks/deliveries":
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			wantQuery := map[string]string{
				"configId":      "config-1",
				"status":        "failed",
				"createdAtFrom": from.Format(time.RFC3339),
				"createdAtTo":   to.Format(time.RFC3339),
				"cursor":        "cursor-1",
				"pageSize":      "25",
			}
			for key, want := range wantQuery {
				if got := r.URL.Query().Get(key); got != want {
					t.Errorf("query %s = %q, want %q", key, got, want)
				}
			}
			_, _ = w.Write([]byte(`{"cursor":{"pageSize":25,"hasMore":true,"next":"cursor-2","data":[{"id":"delivery-1","eventID":"event-1","configID":"config-1","eventType":"ledger.committed_transactions","status":"failed","attemptCount":3,"replayGeneration":0,"createdAt":"2026-07-01T10:00:00Z","updatedAt":"2026-07-01T10:01:00Z"}]}}`))
		case "/api/webhooks/deliveries/delivery-1":
			_, _ = w.Write([]byte(`{"data":{"id":"delivery-1","eventID":"event-1","configID":"config-1","eventType":"ledger.committed_transactions","payload":"{}","status":"failed","attemptCount":3,"replayGeneration":0,"createdAt":"2026-07-01T10:00:00Z","updatedAt":"2026-07-01T10:01:00Z"}}`))
		case "/api/webhooks/deliveries/delivery-1/attempts":
			if got := r.URL.Query().Get("cursor"); got != "attempt-cursor" {
				t.Errorf("cursor = %q, want attempt-cursor", got)
			}
			if got := r.URL.Query().Get("pageSize"); got != "10" {
				t.Errorf("pageSize = %q, want 10", got)
			}
			_, _ = w.Write([]byte(`{"cursor":{"pageSize":10,"hasMore":false,"data":[{"id":"attempt-1","deliveryID":"delivery-1","attemptNumber":1,"replayGeneration":0,"endpoint":"https://example.com/hook","outcome":"permanent_failure","statusCode":404,"durationMillis":12,"createdAt":"2026-07-01T10:01:00Z"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	api := newDeliveriesAPI(server.URL, server.Client())
	list, err := api.list(context.Background(), listDeliveriesParams{
		ConfigID:      "config-1",
		Status:        DeliveryStatusFailed,
		CreatedAtFrom: &from,
		CreatedAtTo:   &to,
		Cursor:        "cursor-1",
		PageSize:      25,
	})
	if err != nil {
		t.Fatalf("list() error = %v", err)
	}
	if len(list.Cursor.Data) != 1 || list.Cursor.Data[0].ID != "delivery-1" || list.Cursor.Next == nil || *list.Cursor.Next != "cursor-2" {
		t.Fatalf("list() response = %#v", list)
	}

	delivery, err := api.get(context.Background(), "delivery-1")
	if err != nil {
		t.Fatalf("get() error = %v", err)
	}
	if delivery.Data.Payload != "{}" {
		t.Fatalf("get() payload = %q, want {}", delivery.Data.Payload)
	}

	attempts, err := api.attempts(context.Background(), "delivery-1", "attempt-cursor", 10)
	if err != nil {
		t.Fatalf("attempts() error = %v", err)
	}
	if len(attempts.Cursor.Data) != 1 || attempts.Cursor.Data[0].StatusCode != 404 || attempts.Cursor.Data[0].DurationMillis != 12 {
		t.Fatalf("attempts() response = %#v", attempts)
	}
}

func TestDeliveriesAPIReplayEndpoints(t *testing.T) {
	from := time.Date(2026, time.July, 1, 10, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	var calls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/webhooks/deliveries/delivery-1/replay":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "individual-key" {
				t.Fatalf("Idempotency-Key = %q", got)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"delivery-1","eventID":"event-1","configID":"config-1","eventType":"event","status":"pending","attemptCount":0,"replayGeneration":1,"createdAt":"2026-07-01T10:00:00Z","updatedAt":"2026-07-01T10:02:00Z"}}`))
		case "/api/webhooks/deliveries/replay":
			if got := r.Header.Get("Idempotency-Key"); got != "bulk-key" {
				t.Fatalf("Idempotency-Key = %q", got)
			}
			var body replayDeliveriesRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if !body.CreatedAtFrom.Equal(from) || body.CreatedAtTo == nil || !body.CreatedAtTo.Equal(to) {
				t.Errorf("window = %s..%v, want %s..%s", body.CreatedAtFrom, body.CreatedAtTo, from, to)
			}
			if !reflect.DeepEqual(body.Statuses, []DeliveryStatus{DeliveryStatusFailed, DeliveryStatusPending}) {
				t.Errorf("statuses = %v", body.Statuses)
			}
			if !reflect.DeepEqual(body.ConfigIDs, []string{"config-1"}) || body.Cursor == nil || *body.Cursor != "cursor-1" || body.PageSize != 1000 {
				t.Errorf("bulk body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"data":{"replayed":2,"expedited":1,"skipped":0,"hasMore":true,"nextCursor":"cursor-2","createdAtTo":"2026-07-02T10:00:00Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	api := newDeliveriesAPI(server.URL, server.Client())
	replayed, err := api.replay(context.Background(), "delivery-1", "individual-key")
	if err != nil {
		t.Fatalf("replay() error = %v", err)
	}
	if replayed.Data.Status != DeliveryStatusPending || replayed.Data.ReplayGeneration != 1 {
		t.Fatalf("replay() response = %#v", replayed)
	}

	cursor := "cursor-1"
	bulk, err := api.replayMany(context.Background(), "bulk-key", replayDeliveriesRequest{
		CreatedAtFrom: from,
		CreatedAtTo:   &to,
		Statuses:      []DeliveryStatus{DeliveryStatusFailed, DeliveryStatusPending},
		ConfigIDs:     []string{"config-1"},
		Cursor:        &cursor,
		PageSize:      1000,
	})
	if err != nil {
		t.Fatalf("replayMany() error = %v", err)
	}
	if bulk.Data.Replayed != 2 || bulk.Data.NextCursor == nil || *bulk.Data.NextCursor != "cursor-2" {
		t.Fatalf("replayMany() response = %#v", bulk)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestDeliveriesAPIIncludesErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"errorCode":"VALIDATION","errorMessage":"delivery cannot be replayed"}`))
	}))
	defer server.Close()

	_, err := newDeliveriesAPI(server.URL, server.Client()).replay(context.Background(), "delivery-1", "key")
	if err == nil {
		t.Fatal("replay() expected error")
	}
	for _, want := range []string{"status 409", "delivery cannot be replayed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want substring %q", err, want)
		}
	}
}

func TestDeliveriesCommandExposesAllEndpoints(t *testing.T) {
	cmd := NewDeliveriesCommand()
	for _, name := range []string{"list", "show", "attempts", "replay", "replay-bulk"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child == cmd || child.Name() != name {
			t.Errorf("command %q not found", name)
		}
	}
	for _, name := range []string{"replay", "replay-bulk"} {
		child, _, _ := cmd.Find([]string{name})
		if child.Flag(idempotencyKeyFlag) == nil {
			t.Errorf("command %q is missing --%s", name, idempotencyKeyFlag)
		}
	}
}

func TestWebhooksCommandExposesConfigEndpoints(t *testing.T) {
	cmd := NewCommand()
	for _, name := range []string{"create", "list", "update", "test", "activate", "deactivate", "delete", "change-secret", "deliveries"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child == cmd || child.Name() != name {
			t.Errorf("command %q not found", name)
		}
	}
}

func TestWebhooksCommandAliasesDoNotConflict(t *testing.T) {
	cmd := NewCommand()

	deleteCommand, _, err := cmd.Find([]string{"del"})
	if err != nil || deleteCommand.Name() != "delete" {
		t.Fatalf("alias del resolved to %q, want delete (err = %v)", deleteCommand.Name(), err)
	}

	deliveriesCommand, _, err := cmd.Find([]string{"dlvs"})
	if err != nil || deliveriesCommand.Name() != "deliveries" {
		t.Fatalf("alias dlvs resolved to %q, want deliveries (err = %v)", deliveriesCommand.Name(), err)
	}
}

func TestDeliveryStatusValidation(t *testing.T) {
	if _, err := deliveryStatus("succeeded", true); err == nil {
		t.Fatal("bulk replay accepted succeeded status")
	}
	if status, err := deliveryStatus("succeeded", false); err != nil || status != DeliveryStatusSucceeded {
		t.Fatalf("list status = %q, err = %v", status, err)
	}
}

func TestReplayCommandsResolveRequiredValuesFromEnvironment(t *testing.T) {
	t.Setenv("IDEMPOTENCY_KEY", "environment-key")
	t.Setenv("CREATED_AT_FROM", "2026-08-01T10:00:00Z")

	individual := NewReplayDeliveryCommand()
	key, err := requiredIdempotencyKey(individual)
	if err != nil || key != "environment-key" {
		t.Fatalf("idempotency key = %q, err = %v", key, err)
	}

	bulk := NewReplayDeliveriesCommand()
	from, err := fctl.GetDateTime(bulk, createdAtFromFlag)
	if err != nil || from == nil || from.Format(time.RFC3339) != "2026-08-01T10:00:00Z" {
		t.Fatalf("created-at-from = %v, err = %v", from, err)
	}
}

func TestRequiredIdempotencyKeyRejectsMissingValue(t *testing.T) {
	t.Setenv("IDEMPOTENCY_KEY", "")
	_, err := requiredIdempotencyKey(NewReplayDeliveryCommand())
	if err == nil {
		t.Fatal("requiredIdempotencyKey() expected an error")
	}
}

func TestReplayStatusesUseEnvironmentBeforeDefaults(t *testing.T) {
	t.Setenv("STATUS", "failed")
	statuses, err := resolvedReplayStatuses(NewReplayDeliveriesCommand())
	if err != nil {
		t.Fatalf("resolvedReplayStatuses() error = %v", err)
	}
	if !reflect.DeepEqual(statuses, []DeliveryStatus{DeliveryStatusFailed}) {
		t.Fatalf("statuses = %v, want [failed]", statuses)
	}
}

func TestReplayStatusesDefaultToFailedAndPending(t *testing.T) {
	t.Setenv("STATUS", "")
	statuses, err := resolvedReplayStatuses(NewReplayDeliveriesCommand())
	if err != nil {
		t.Fatalf("resolvedReplayStatuses() error = %v", err)
	}
	want := []DeliveryStatus{DeliveryStatusFailed, DeliveryStatusPending}
	if !reflect.DeepEqual(statuses, want) {
		t.Fatalf("statuses = %v, want %v", statuses, want)
	}
}
