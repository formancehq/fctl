package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

// Some historical servers return before the persisted instance is terminal,
// even with wait=true. Read its state without starting another workflow.
func runWorkflow(ctx context.Context, client *httpclient.Client, request pluginsdk.ExecuteRequest, op operation, body json.RawMessage) (json.RawMessage, error) {
	data, err := perform(ctx, client, request, op, body)
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
	if identifier(state.ID) != nil || state.Terminated == nil || (expectedID != "" && state.ID != expectedID) {
		return "", false, fmt.Errorf("workflow instance response must contain the requested id and terminated state")
	}
	if state.Error != "" {
		return state.ID, false, fmt.Errorf("workflow instance %s: %s", state.ID, state.Error)
	}
	return state.ID, *state.Terminated, nil
}
