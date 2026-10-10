package modules_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/modules"
)

func TestWorkflowWaitReadsTerminalState(t *testing.T) {
	t.Parallel()
	runCase(t, commandCase{command: "orchestration workflows run", args: []string{"flow"}, flags: map[string]string{"wait": "true"}, exchanges: []exchange{
		{method: "POST", path: "/workflows/flow/instances", query: "wait=true", body: `{}`, response: `{"data":{"id":"inst/a","terminated":false}}`},
		{method: "GET", path: "/instances/inst%2Fa", response: `{"data":{"id":"inst/a","terminated":false}}`},
		{method: "GET", path: "/instances/inst%2Fa", response: `{"data":{"id":"inst/a","terminated":true,"amount":9007199254740993}}`},
	}})
}

func TestWorkflowWaitRefusesInvalidState(t *testing.T) {
	t.Parallel()
	for _, response := range []string{
		`{"data":{"id":"inst"}}`,
		`{"data":{"terminated":false}}`,
		`{"data":{"id":"inst","terminated":false,"error":"workflow failed"}}`,
	} {
		t.Run(response, func(t *testing.T) {
			t.Parallel()
			_, err := executeWait(t.Context(), t, []exchange{workflowStart(response)})
			if err == nil {
				t.Fatal("accepted incomplete or failed workflow")
			}
		})
	}
}

func TestWorkflowWaitCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := executeWait(ctx, t, []exchange{workflowStart(`{"data":{"id":"inst","terminated":false}}`)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting did not preserve cancellation: %v", err)
	}
}

func TestWorkflowWaitReadFailure(t *testing.T) {
	t.Parallel()
	_, err := executeWait(t.Context(), t, []exchange{
		workflowStart(`{"data":{"id":"inst","terminated":false}}`),
		{method: http.MethodGet, path: "/instances/inst", status: http.StatusServiceUnavailable, response: `{"errorMessage":"unavailable"}`},
	})
	var failure *httpclient.Error
	if !errors.As(err, &failure) || failure.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("read failure lost: %v", err)
	}
}

func TestWorkflowWaitRefusesAnotherInstance(t *testing.T) {
	t.Parallel()
	_, err := executeWait(t.Context(), t, []exchange{
		workflowStart(`{"data":{"id":"inst","terminated":false}}`),
		{method: http.MethodGet, path: "/instances/inst", response: `{"data":{"id":"another","terminated":true}}`},
	})
	if err == nil || !strings.Contains(err.Error(), "requested id") {
		t.Fatalf("accepted another instance: %v", err)
	}
}

func workflowStart(response string) exchange {
	return exchange{method: http.MethodPost, path: "/workflows/flow/instances", query: "wait=true", body: `{}`, response: response}
}

func executeWait(ctx context.Context, t *testing.T, exchanges []exchange) (pluginsdk.ExecuteResponse, error) {
	t.Helper()
	server, verify := fixture(t, exchanges)
	defer verify()
	p := modules.Factories()["orchestration"](server.Client())
	return p.Execute(ctx, pluginsdk.ExecuteRequest{CommandPath: []string{"orchestration", "workflows", "run"}, Args: []string{"flow"}, Flags: map[string]string{"wait": "true"}, Endpoint: server.URL + "/gateway/service"})
}
