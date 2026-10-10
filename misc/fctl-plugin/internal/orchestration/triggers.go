package orchestration

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func triggerOperations() []command.Operation {
	trigger := command.Leaf("triggers create", "create <event> <workflow-id>", http.MethodPost, "triggers", 2, 2,
		command.StringFlag("name", "Trigger name"), command.StringFlag("filter", "Event filter expression"), command.StringFlag("vars", "Variables as JSON object or CSV key=value"))
	trigger.Body = triggerCreateBody
	triggers := command.Leaf("triggers list", "list", http.MethodGet, "triggers", 0, 0, command.StringFlag("name", "Trigger name filter"))
	triggers.Query = map[string]string{"name": "name"}
	test := command.Leaf("triggers test", "test <trigger-id> [event-json]", http.MethodPost, "v2/triggers/$0/test", 1, 2, command.DataFlag())
	test.Body = triggerTestBody
	test.Spec.Inputs = append(test.Spec.Inputs, pluginsdk.InputSpec{Title: "Event JSON", Kind: "text", Flag: "data", Required: true, AlternativeArgument: new(1)})
	return []command.Operation{
		triggers, trigger, test,
		command.Leaf("triggers show", "show <trigger-id>", http.MethodGet, "triggers/$0", 1, 1),
		command.Confirmed(command.Leaf("triggers delete", "delete <trigger-id>", http.MethodDelete, "triggers/$0", 1, 1)),
		command.Leaf("triggers occurrences list", "list <trigger-id>", http.MethodGet, "triggers/$0/occurrences", 1, 1),
	}
}

func triggerCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	for _, v := range r.Args {
		if err := command.Identifier(v); err != nil {
			return nil, err
		}
	}
	vars, err := command.Pairs(r.Flags["vars"])
	if err != nil {
		return nil, err
	}
	body := map[string]any{"event": r.Args[0], "workflowID": r.Args[1], "name": r.Flags["name"], "vars": vars}
	if r.Flags["filter"] != "" {
		body["filter"] = r.Flags["filter"]
	}
	return json.Marshal(body)
}

func triggerTestBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil && len(r.Args) == 2 {
		return nil, fmt.Errorf("provide event JSON once, as an argument or --data")
	}
	if len(r.Args) == 2 {
		return command.ObjectBody(json.RawMessage(r.Args[1]))
	}
	return command.ObjectBody(r.Body)
}
