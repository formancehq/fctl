package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func newOrchestration(client *http.Client) pluginsdk.Plugin {
	instances := leaf("instances list", "list", http.MethodGet, "instances", 0, 0, str("workflow", "Workflow ID filter"), boolean("running", "Only running instances"))
	instances.query = map[string]string{"workflow": "workflowID", "running": "running"}
	show := leaf("instances show", "show <instance-id>", http.MethodGet, "instances/$0", 1, 1)
	show.run = instanceWithWorkflow
	describe := leaf("instances describe", "describe <instance-id>", http.MethodGet, "instances/$0/history", 1, 1)
	describe.run = instanceHistory
	event := leaf("instances send-event", "send-event <instance-id> <event>", http.MethodPost, "instances/$0/events", 2, 2)
	event.body = eventBody
	workflowCreate := payload(leaf("workflows create", "create", http.MethodPost, "workflows", 0, 0), "stages")
	workflowCreate.body = workflowCreateBody
	run := leaf("workflows run", "run <workflow-id>", http.MethodPost, "workflows/$0/instances", 1, 1, str("variable", "Variables as JSON object or CSV key=value"), boolean("wait", "Wait for completion"), dataFlag())
	run.query = map[string]string{"wait": "wait"}
	run.body = workflowRunBody
	trigger := leaf("triggers create", "create <event> <workflow-id>", http.MethodPost, "triggers", 2, 2,
		str("name", "Trigger name"), str("filter", "Event filter expression"), str("vars", "Variables as JSON object or CSV key=value"))
	trigger.body = triggerCreateBody
	triggers := leaf("triggers list", "list", http.MethodGet, "triggers", 0, 0, str("name", "Trigger name filter"))
	triggers.query = map[string]string{"name": "name"}
	test := leaf("triggers test", "test <trigger-id> [event-json]", http.MethodPost, "v2/triggers/$0/test", 1, 2, dataFlag())
	test.body = triggerTestBody
	test.spec.Inputs = append(test.spec.Inputs, pluginsdk.InputSpec{Title: "Event JSON", Kind: "text", Flag: "data", Required: true, AlternativeArgument: new(1)})
	return newPlugin("orchestration", client, []operation{
		instances, show, describe, event,
		confirmed(leaf("instances stop", "stop <instance-id>", http.MethodPut, "instances/$0/abort", 1, 1)),
		leaf("workflows list", "list", http.MethodGet, "workflows", 0, 0),
		leaf("workflows show", "show <workflow-id>", http.MethodGet, "workflows/$0", 1, 1), workflowCreate, run,
		confirmed(leaf("workflows delete", "delete <workflow-id>", http.MethodDelete, "workflows/$0", 1, 1)),
		triggers, trigger, test,
		leaf("triggers show", "show <trigger-id>", http.MethodGet, "triggers/$0", 1, 1),
		confirmed(leaf("triggers delete", "delete <trigger-id>", http.MethodDelete, "triggers/$0", 1, 1)),
		leaf("triggers occurrences list", "list <trigger-id>", http.MethodGet, "triggers/$0/occurrences", 1, 1),
	})
}

func instanceWithWorkflow(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op operation, body json.RawMessage) (json.RawMessage, error) {
	instance, err := perform(ctx, client, r, op, body)
	if err != nil {
		return instance, err
	}
	var envelope struct {
		Data struct {
			WorkflowID string `json:"workflowID"`
		} `json:"data"`
	}
	if json.Unmarshal(instance, &envelope) != nil || identifier(envelope.Data.WorkflowID) != nil {
		return nil, fmt.Errorf("instance response is missing data.workflowID")
	}
	workflow, err := client.Do(ctx, http.MethodGet, httpclient.Path("workflows", envelope.Data.WorkflowID), nil, nil, nil)
	if err != nil {
		return instance, err
	}
	return marshal(map[string]json.RawMessage{"instance": instance, "workflow": workflow})
}

func instanceHistory(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op operation, body json.RawMessage) (json.RawMessage, error) {
	history, err := perform(ctx, client, r, op, body)
	if err != nil {
		return history, err
	}
	var envelope struct {
		Data []struct {
			Stage struct {
				Type string `json:"type"`
			} `json:"stage"`
		} `json:"data"`
	}
	if json.Unmarshal(history, &envelope) != nil || envelope.Data == nil {
		return nil, fmt.Errorf("instance history response is missing a data array")
	}
	stages := map[string]json.RawMessage{}
	for i, entry := range envelope.Data {
		if entry.Stage.Type != "send" {
			continue
		}
		stage, err := client.Do(ctx, http.MethodGet, httpclient.Path("instances", r.Args[0], "stages", strconv.Itoa(i), "history"), nil, nil, nil)
		if err != nil {
			return history, err
		}
		stages[strconv.Itoa(i)] = stage
	}
	return marshal(map[string]any{"history": history, "stages": stages})
}

func eventBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := identifier(r.Args[1]); err != nil {
		return nil, fmt.Errorf("event: %w", err)
	}
	return marshal(map[string]string{"name": r.Args[1]})
}

func workflowCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	body, err := objectBody(r.Body, "stages")
	if err != nil {
		return nil, err
	}
	var config struct {
		Stages []json.RawMessage `json:"stages"`
	}
	if json.Unmarshal(body, &config) != nil || len(config.Stages) == 0 {
		return nil, fmt.Errorf("workflow stages must be a nonempty array")
	}
	for _, stage := range config.Stages {
		if _, err := objectBody(stage); err != nil {
			return nil, fmt.Errorf("invalid workflow stage: %w", err)
		}
	}
	return body, nil
}

func workflowRunBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil {
		return objectBody(r.Body)
	}
	vars, err := pairs(r.Flags["variable"])
	if err != nil {
		return nil, err
	}
	return marshal(vars)
}

func triggerCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	for _, v := range r.Args {
		if err := identifier(v); err != nil {
			return nil, err
		}
	}
	vars, err := pairs(r.Flags["vars"])
	if err != nil {
		return nil, err
	}
	body := map[string]any{"event": r.Args[0], "workflowID": r.Args[1], "name": r.Flags["name"], "vars": vars}
	if r.Flags["filter"] != "" {
		body["filter"] = r.Flags["filter"]
	}
	return marshal(body)
}

func triggerTestBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil && len(r.Args) == 2 {
		return nil, fmt.Errorf("provide event JSON once, as an argument or --data")
	}
	if len(r.Args) == 2 {
		return objectBody(json.RawMessage(r.Args[1]))
	}
	return objectBody(r.Body)
}
