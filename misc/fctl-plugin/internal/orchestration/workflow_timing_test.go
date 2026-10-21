package orchestration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func TestWorkflowWaitPollInterval(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		transport := &workflowClockTransport{t: t, terminalAfter: 2}
		started := time.Now()
		response, err := New(&http.Client{Transport: transport}).Execute(t.Context(), waitRequest())
		if err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(started); elapsed != time.Second || transport.polls != 2 {
			t.Fatalf("polled %d times in %s; expected two polls in 1s", transport.polls, elapsed)
		}
		if string(response.Data) != `{"data":{"id":"inst/a","terminated":true,"amount":9007199254740993}}` {
			t.Fatalf("lost terminal response: %s", response.Data)
		}
	})
}

func TestWorkflowWaitFiveMinuteDeadline(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// Response time separates the read timer from the deadline so this tests
		// cancellation while waiting, rather than two simultaneous timer events.
		transport := &workflowClockTransport{t: t, readLatency: time.Millisecond}
		started := time.Now()
		response, err := New(&http.Client{Transport: transport}).Execute(t.Context(), waitRequest())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("lost deadline: %v", err)
		}
		if elapsed := time.Since(started); elapsed != 5*time.Minute {
			t.Fatalf("wait deadline is %s; expected 5m", elapsed)
		}
		if err.Error() != "waiting for workflow instance inst/a: context deadline exceeded" {
			t.Fatalf("lost instance context: %v", err)
		}
		if string(response.Data) != `{"data":{"id":"inst/a","terminated":false,"amount":9007199254740993}}` {
			t.Fatalf("lost last workflow response: %s", response.Data)
		}
	})
}

func waitRequest() pluginsdk.ExecuteRequest {
	return pluginsdk.ExecuteRequest{
		CommandPath: []string{"orchestration", "workflows", "run"},
		Args:        []string{"flow"}, Flags: map[string]string{"wait": "true"},
		Endpoint: "https://example.invalid/gateway/service",
	}
}

// This transport stays inside the virtual clock so the five-minute deadline
// can be checked without real time or an external server.
type workflowClockTransport struct {
	t             *testing.T
	terminalAfter int
	readLatency   time.Duration
	polls         int
	lastResponse  time.Time
}

func (c *workflowClockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.t.Helper()
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if r.Method == http.MethodPost {
		if c.polls != 0 || !c.lastResponse.IsZero() || r.URL.EscapedPath() != "/gateway/service/workflows/flow/instances" || r.URL.RawQuery != "wait=true" {
			c.t.Fatalf("unexpected workflow start: %s %s", r.Method, r.URL)
		}
	} else {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/gateway/service/instances/inst%2Fa" || r.URL.RawQuery != "" {
			c.t.Fatalf("unexpected workflow poll: %s %s", r.Method, r.URL)
		}
		if interval := time.Since(c.lastResponse); interval != 500*time.Millisecond {
			c.t.Fatalf("poll interval is %s; expected 500ms", interval)
		}
		c.polls++
		time.Sleep(c.readLatency)
	}
	c.lastResponse = time.Now()
	terminal := c.terminalAfter > 0 && c.polls >= c.terminalAfter
	body := fmt.Sprintf(`{"data":{"id":"inst/a","terminated":%t,"amount":9007199254740993}}`, terminal)
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
}
