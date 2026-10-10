package reconciliation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func alertOperations() []command.Operation {
	operations := []command.Operation{
		command.Leaf("alerts get", "get <alert-id>", http.MethodGet, "alerts/$0", 1, 1),
		paginatedList("alerts events", "events <alert-id>", "alerts/$0/events", 1),
		clarityList("alerts"),
	}
	for _, transition := range []string{"ack", "resolve", "accept", "snooze", "unsnooze"} {
		by := command.StringFlag("by", "Actor")
		by.Required = true
		flags := []pluginsdk.FlagSpec{by}
		if transition != "unsnooze" {
			note := command.StringFlag("note", "Note or business justification")
			note.Required = transition == "accept"
			flags = append(flags, note)
		}
		if transition == "resolve" {
			flags = append(flags, command.StringFlag("transaction-ref", "Transaction references as CSV or JSON string array"))
		}
		if transition == "snooze" {
			until := command.StringFlag("until", "Future RFC3339 deadline")
			until.Required = true
			flags = append(flags, until)
		}
		op := command.Confirmed(command.Leaf("alerts "+transition, transition+" <alert-id>", http.MethodPost, "alerts/$0/"+transition, 1, 1, flags...))
		op.Body = alertTransition
		for _, f := range flags {
			if f.Required {
				op.Spec.Inputs = append(op.Spec.Inputs, pluginsdk.InputSpec{Title: f.Usage, Kind: "input", Flag: f.Name, Required: true})
			}
		}
		operations = append(operations, op)
	}
	return operations
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
		refs, err := command.List(v)
		if err != nil {
			return nil, err
		}
		body["transactionRefs"] = refs
	}
	return json.Marshal(body)
}
