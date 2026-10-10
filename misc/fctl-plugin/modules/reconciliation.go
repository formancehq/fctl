package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func newReconciliation(client *http.Client) pluginsdk.Plugin {
	operations := []operation{
		leaf("get", "get <reconciliation-id>", http.MethodGet, "reconciliations/$0", 1, 1),
		leaf("policies get", "get <policy-id>", http.MethodGet, "policies/$0", 1, 1),
		confirmed(payload(leaf("policies create", "create", http.MethodPost, "policies", 0, 0), "name", "ledgerName", "paymentsPoolID", "ledgerQuery")),
		confirmed(leaf("policies delete", "delete <policy-id>", http.MethodDelete, "policies/$0", 1, 1)),
		leaf("rules get", "get <rule-id>", http.MethodGet, "rules/$0", 1, 1),
		confirmed(payload(leaf("rules create", "create", http.MethodPost, "rules", 0, 0))),
		confirmed(payload(leaf("rules update", "update <rule-id>", http.MethodPatch, "rules/$0", 1, 1))),
		confirmed(leaf("rules delete", "delete <rule-id>", http.MethodDelete, "rules/$0", 1, 1)),
		leaf("evaluations get", "get <evaluation-id>", http.MethodGet, "evaluations/$0", 1, 1),
		leaf("alerts get", "get <alert-id>", http.MethodGet, "alerts/$0", 1, 1),
	}
	for _, pair := range [][2]string{{"list", "reconciliations"}, {"policies list", "policies"}, {"alerts events", "alerts/$0/events"}} {
		args := 0
		use := "list"
		if pair[0] == "alerts events" {
			args = 1
			use = "events <alert-id>"
		}
		op := leaf(pair[0], use, http.MethodGet, pair[1], args, args, pagination()...)
		op.query = map[string]string{"cursor": "cursor", "page-size": "pageSize"}
		op.run = cursorOperation
		operations = append(operations, op)
	}
	for _, group := range []string{"rules", "evaluations", "alerts"} {
		op := leaf(group+" list", "list", http.MethodGet, group, 0, 0, append(pagination(), str("query", "JSON query-builder expression"), dataFlag())...)
		op.query = map[string]string{"cursor": "cursor", "page-size": "pageSize"}
		op.body = clarityListBody
		op.run = cursorOperation
		operations = append(operations, op)
	}
	reconcile := leaf("policies reconcile", "reconcile <policy-id> <at-ledger> <at-payments>", http.MethodPost, "policies/$0/reconciliation", 3, 3)
	reconcile.body = reconcileBody
	evaluate := confirmed(leaf("rules evaluate", "evaluate <rule-id>", http.MethodPost, "rules/$0/evaluate", 1, 1,
		str("at", "Canonical RFC3339 point in time"), str("safety-margin", "Nonnegative Go duration, e.g. 30s"), str("source-pit", "JSON object or CSV source=RFC3339")))
	evaluate.body = evaluateRuleBody
	operations = append(operations, reconcile, evaluate)
	operations = append(operations, alertOperations()...)
	return newPlugin("reconciliation", client, operations)
}

func cursorOperation(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op operation, body json.RawMessage) (json.RawMessage, error) {
	query, err := queryValues(op.query, r)
	if err != nil {
		return nil, err
	}
	if cursor := query.Get("cursor"); cursor != "" {
		query = url.Values{"cursor": {cursor}}
		body = nil
	}
	path, err := route(op.segments, r)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, op.method, path, query, body, nil)
}

func alertTransition(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if strings.TrimSpace(r.Flags["by"]) == "" {
		return nil, fmt.Errorf("--by must be nonempty")
	}
	body := map[string]any{"by": r.Flags["by"]}
	if note := r.Flags["note"]; note != "" {
		body["note"] = note
	}
	if r.CommandPath[2] == "accept" && strings.TrimSpace(r.Flags["note"]) == "" {
		return nil, fmt.Errorf("--note is required for accept")
	}
	if v := r.Flags["until"]; v != "" {
		deadline, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return nil, fmt.Errorf("--until must be RFC3339: %w", err)
		}
		if !deadline.After(time.Now()) {
			return nil, fmt.Errorf("--until must be in the future")
		}
		body["until"] = v
	}
	if v := r.Flags["transaction-ref"]; v != "" {
		refs, err := list(v)
		if err != nil {
			return nil, err
		}
		body["transactionRefs"] = refs
	}
	return marshal(body)
}

func clarityListBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Flags["cursor"] != "" {
		return nil, nil
	}
	if r.Body != nil && r.Flags["query"] != "" {
		return nil, fmt.Errorf("provide --query or --data once")
	}
	if r.Body != nil {
		return objectBody(r.Body)
	}
	if r.Flags["query"] != "" {
		return objectBody(json.RawMessage(r.Flags["query"]))
	}
	return nil, nil
}

func reconcileBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	for _, value := range r.Args[1:] {
		if err := timestamp(value); err != nil {
			return nil, err
		}
	}
	return marshal(map[string]string{"reconciledAtLedger": r.Args[1], "reconciledAtPayments": r.Args[2]})
}

func evaluateRuleBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	body := map[string]any{}
	if v := r.Flags["at"]; v != "" {
		if err := timestamp(v); err != nil {
			return nil, err
		}
		body["at"] = v
	}
	if v := r.Flags["safety-margin"]; v != "" {
		duration, err := time.ParseDuration(v)
		if err != nil || duration < 0 {
			return nil, fmt.Errorf("--safety-margin must be a nonnegative duration")
		}
		body["safetyMargin"] = v
	}
	if v := r.Flags["source-pit"]; v != "" {
		values, err := pairs(v)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			if err := timestamp(value); err != nil {
				return nil, err
			}
		}
		body["sourcePITs"] = values
	}
	return marshal(body)
}

func alertOperations() []operation {
	operations := []operation{}
	for _, transition := range []string{"ack", "resolve", "accept", "snooze", "unsnooze"} {
		by := str("by", "Actor")
		by.Required = true
		flags := []pluginsdk.FlagSpec{by}
		if transition != "unsnooze" {
			note := str("note", "Note or business justification")
			note.Required = transition == "accept"
			flags = append(flags, note)
		}
		if transition == "resolve" {
			flags = append(flags, str("transaction-ref", "Transaction references as CSV or JSON string array"))
		}
		if transition == "snooze" {
			until := str("until", "Future RFC3339 deadline")
			until.Required = true
			flags = append(flags, until)
		}
		op := confirmed(leaf("alerts "+transition, transition+" <alert-id>", http.MethodPost, "alerts/$0/"+transition, 1, 1, flags...))
		op.body = alertTransition
		for _, f := range flags {
			if f.Required {
				op.spec.Inputs = append(op.spec.Inputs, pluginsdk.InputSpec{Title: f.Usage, Kind: "input", Flag: f.Name, Required: true})
			}
		}
		operations = append(operations, op)
	}
	return operations
}
