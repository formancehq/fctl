package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func instanceOperations() []command.Operation {
	instances := command.Leaf("instances list", "list", http.MethodGet, "instances", 0, 0, command.StringFlag("workflow", "Workflow ID filter"), command.BoolFlag("running", "Only running instances"))
	instances.Query = map[string]string{"workflow": "workflowID", "running": "running"}
	show := command.Leaf("instances show", "show <instance-id>", http.MethodGet, "instances/$0", 1, 1)
	show.Run = instanceWithWorkflow
	describe := command.Leaf("instances describe", "describe <instance-id>", http.MethodGet, "instances/$0/history", 1, 1)
	describe.Run = instanceHistory
	event := command.Leaf("instances send-event", "send-event <instance-id> <event>", http.MethodPost, "instances/$0/events", 2, 2)
	event.Body = eventBody
	return []command.Operation{
		instances, show, describe, event,
		command.Confirmed(command.Leaf("instances stop", "stop <instance-id>", http.MethodPut, "instances/$0/abort", 1, 1)),
	}
}

func instanceWithWorkflow(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op command.Operation, body json.RawMessage) (json.RawMessage, error) {
	instance, err := command.Perform(ctx, client, r, op, body)
	if err != nil {
		return instance, err
	}
	var envelope struct {
		Data struct {
			WorkflowID string `json:"workflowID"`
		} `json:"data"`
	}
	if json.Unmarshal(instance, &envelope) != nil || command.Identifier(envelope.Data.WorkflowID) != nil {
		return nil, fmt.Errorf("instance response is missing data.workflowID")
	}
	workflow, err := client.Do(ctx, http.MethodGet, httpclient.Path("workflows", envelope.Data.WorkflowID), nil, nil, nil)
	if err != nil {
		return instance, err
	}
	return json.Marshal(map[string]json.RawMessage{"instance": instance, "workflow": workflow})
}

func instanceHistory(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op command.Operation, body json.RawMessage) (json.RawMessage, error) {
	history, err := command.Perform(ctx, client, r, op, body)
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
	return json.Marshal(map[string]any{"history": history, "stages": stages})
}

func eventBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := command.Identifier(r.Args[1]); err != nil {
		return nil, fmt.Errorf("event: %w", err)
	}
	return json.Marshal(map[string]string{"name": r.Args[1]})
}
