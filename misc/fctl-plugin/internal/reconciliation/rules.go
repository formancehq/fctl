package reconciliation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func ruleOperations() []command.Operation {
	evaluate := command.Confirmed(command.Leaf("rules evaluate", "evaluate <rule-id>", http.MethodPost, "rules/$0/evaluate", 1, 1,
		command.StringFlag("at", "Canonical RFC3339 point in time"), command.StringFlag("safety-margin", "Nonnegative Go duration, e.g. 30s"), command.StringFlag("source-pit", "JSON object or CSV source=RFC3339")))
	evaluate.Body = evaluateRuleBody
	return []command.Operation{
		command.Leaf("rules get", "get <rule-id>", http.MethodGet, "rules/$0", 1, 1),
		command.Confirmed(command.Payload(command.Leaf("rules create", "create", http.MethodPost, "rules", 0, 0))),
		command.Confirmed(command.Payload(command.Leaf("rules update", "update <rule-id>", http.MethodPatch, "rules/$0", 1, 1))),
		command.Confirmed(command.Leaf("rules delete", "delete <rule-id>", http.MethodDelete, "rules/$0", 1, 1)),
		clarityList("rules"),
		evaluate,
	}
}

func evaluateRuleBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	body := map[string]any{}
	if v := r.Flags["at"]; v != "" {
		if err := command.Timestamp(v); err != nil {
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
		values, err := command.Pairs(v)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			if err := command.Timestamp(value); err != nil {
				return nil, err
			}
		}
		body["sourcePITs"] = values
	}
	return json.Marshal(body)
}
