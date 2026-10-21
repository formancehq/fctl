package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestWorkflowWaitReadsTerminalState(t *testing.T) {
	t.Parallel()
	terminal := `{"data":{"id":"inst/a","terminated":true,"amount":9007199254740993}}`
	response, err := executeWait(t.Context(), t, []testutil.Exchange{
		{Method: "POST", Path: "/workflows/flow/instances", Query: "wait=true", Body: `{}`, Response: `{"data":{"id":"inst/a","terminated":false}}`},
		{Method: "GET", Path: "/instances/inst%2Fa", Response: `{"data":{"id":"inst/a","terminated":false}}`},
		{Method: "GET", Path: "/instances/inst%2Fa", Response: terminal},
	})
	if err != nil || string(response.Data) != terminal {
		t.Fatalf("terminal response lost: %s, %v", response.Data, err)
	}
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
			_, err := executeWait(t.Context(), t, []testutil.Exchange{workflowStart(response)})
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
	state := `{"data":{"id":"inst","terminated":false}}`
	response, err := executeWait(ctx, t, []testutil.Exchange{workflowStart(state)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting did not preserve cancellation: %v", err)
	}
	if err.Error() != "waiting for workflow instance inst: context deadline exceeded" || string(response.Data) != state {
		t.Fatalf("waiting lost its instance context: %s, %v", response.Data, err)
	}
}

func TestWorkflowWaitReadFailure(t *testing.T) {
	t.Parallel()
	failureBody := `{"errorMessage":"unavailable","amount":9007199254740993}`
	response, err := executeWait(t.Context(), t, []testutil.Exchange{
		workflowStart(`{"data":{"id":"inst","terminated":false}}`),
		{Method: http.MethodGet, Path: "/instances/inst", Status: http.StatusServiceUnavailable, Response: failureBody},
	})
	var failure *httpclient.Error
	if !errors.As(err, &failure) || failure.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("read failure lost: %v", err)
	}
	if string(response.Data) != failureBody {
		t.Fatalf("lost service failure body: %s", response.Data)
	}
}

func TestWorkflowWaitReportsPolledFailure(t *testing.T) {
	t.Parallel()
	state := `{"data":{"id":"inst","terminated":true,"error":"workflow failed","amount":9007199254740993}}`
	response, err := executeWait(t.Context(), t, []testutil.Exchange{
		workflowStart(`{"data":{"id":"inst","terminated":false}}`),
		{Method: http.MethodGet, Path: "/instances/inst", Response: state},
	})
	if err == nil || err.Error() != "workflow instance inst: workflow failed" || string(response.Data) != state {
		t.Fatalf("workflow failure lost: %s, %v", response.Data, err)
	}
}

func TestWorkflowWaitRefusesAnotherInstance(t *testing.T) {
	t.Parallel()
	_, err := executeWait(t.Context(), t, []testutil.Exchange{
		workflowStart(`{"data":{"id":"inst","terminated":false}}`),
		{Method: http.MethodGet, Path: "/instances/inst", Response: `{"data":{"id":"another","terminated":true}}`},
	})
	if err == nil || !strings.Contains(err.Error(), "requested id") {
		t.Fatalf("accepted another instance: %v", err)
	}
}

func workflowStart(response string) testutil.Exchange {
	return testutil.Exchange{Method: http.MethodPost, Path: "/workflows/flow/instances", Query: "wait=true", Body: `{}`, Response: response}
}

func executeWait(ctx context.Context, t *testing.T, exchanges []testutil.Exchange) (pluginsdk.ExecuteResponse, error) {
	t.Helper()
	server, verify := testutil.Fixture(t, exchanges)
	defer verify()
	p := New(server.Client())
	return p.Execute(ctx, pluginsdk.ExecuteRequest{CommandPath: []string{"orchestration", "workflows", "run"}, Args: []string{"flow"}, Flags: map[string]string{"wait": "true"}, Endpoint: server.URL + "/gateway/service"})
}
