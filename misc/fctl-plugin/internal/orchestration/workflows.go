package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func workflowOperations() []command.Operation {
	workflowCreate := command.Payload(command.Leaf("workflows create", "create", http.MethodPost, "workflows", 0, 0), "stages")
	workflowCreate.Body = workflowCreateBody
	run := command.Leaf("workflows run", "run <workflow-id>", http.MethodPost, "workflows/$0/instances", 1, 1, command.StringFlag("variable", "Variables as JSON object or CSV key=value"), command.BoolFlag("wait", "Wait for completion"), command.DataFlag())
	run.Query = map[string]string{"wait": "wait"}
	run.Body = workflowRunBody
	run.Run = runWorkflow
	return []command.Operation{
		command.Leaf("workflows list", "list", http.MethodGet, "workflows", 0, 0),
		command.Leaf("workflows show", "show <workflow-id>", http.MethodGet, "workflows/$0", 1, 1), workflowCreate, run,
		command.Confirmed(command.Leaf("workflows delete", "delete <workflow-id>", http.MethodDelete, "workflows/$0", 1, 1)),
	}
}

func workflowCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	body, err := command.ObjectBody(r.Body, "stages")
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
		if _, err := command.ObjectBody(stage); err != nil {
			return nil, fmt.Errorf("invalid workflow stage: %w", err)
		}
	}
	return body, nil
}

func workflowRunBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil {
		return command.ObjectBody(r.Body)
	}
	vars, err := command.Pairs(r.Flags["variable"])
	if err != nil {
		return nil, err
	}
	return json.Marshal(vars)
}

// Some historical servers return before the persisted instance is terminal,
// even with wait=true. Read its state without starting another workflow.
func runWorkflow(ctx context.Context, client *httpclient.Client, request pluginsdk.ExecuteRequest, op command.Operation, body json.RawMessage) (json.RawMessage, error) {
	data, err := command.Perform(ctx, client, request, op, body)
	if err != nil || request.Flags["wait"] != "true" {
		return data, err
	}
	id, terminated, err := workflowState(data, "")
	if err != nil || terminated {
		return data, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	for {
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return data, fmt.Errorf("waiting for workflow instance %s: %w", id, ctx.Err())
		case <-timer.C:
		}
		data, err = client.Do(ctx, http.MethodGet, httpclient.Path("instances", id), nil, nil, nil)
		if err != nil {
			return data, err
		}
		_, terminated, err = workflowState(data, id)
		if err != nil || terminated {
			return data, err
		}
	}
}

func workflowState(data json.RawMessage, expectedID string) (string, bool, error) {
	var envelope struct {
		Data struct {
			ID         string `json:"id"`
			Terminated *bool  `json:"terminated"`
			Error      string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", false, fmt.Errorf("invalid workflow instance response: %w", err)
	}
	state := envelope.Data
	if command.Identifier(state.ID) != nil || state.Terminated == nil || (expectedID != "" && state.ID != expectedID) {
		return "", false, fmt.Errorf("workflow instance response must contain the requested id and terminated state")
	}
	if state.Error != "" {
		return state.ID, false, fmt.Errorf("workflow instance %s: %s", state.ID, state.Error)
	}
	return state.ID, *state.Terminated, nil
}
