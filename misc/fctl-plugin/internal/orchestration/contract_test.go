package orchestration

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestOrchestrationContracts(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "workflows list", Command: "orchestration workflows list", Exchanges: []testutil.Exchange{{Method: "GET", Path: "/workflows"}}},
		{Name: "workflow show", Command: "orchestration workflows show", Args: []string{"flow/a"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/workflows/flow%2Fa"}}},
		{Name: "workflow create", Command: "orchestration workflows create", Body: `{"name":"flow","stages":[{"send":{"amount":9007199254740993}}]}`, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/workflows", Body: `{"name":"flow","stages":[{"send":{"amount":9007199254740993}}]}`}}},
		{Name: "workflow run", Command: "orchestration workflows run", Args: []string{"flow"}, Flags: map[string]string{"wait": "true", "variable": "user=a=1,asset=USD/2"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/workflows/flow/instances", Query: "wait=true", Body: `{"user":"a=1","asset":"USD/2"}`, Response: `{"data":{"id":"inst","terminated":true}}`}}},
		{Name: "workflow run raw", Command: "orchestration workflows run", Args: []string{"flow"}, Body: `{"quantity":9007199254740993}`, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/workflows/flow/instances", Query: "wait=false", Body: `{"quantity":9007199254740993}`}}},
		{Name: "workflow delete", Command: "orchestration workflows delete", Args: []string{"flow"}, Flags: map[string]string{"confirm": "true"}, Exchanges: []testutil.Exchange{{Method: "DELETE", Path: "/workflows/flow"}}},
		{Name: "instances list", Command: "orchestration instances list", Flags: map[string]string{"workflow": "flow", "running": "true"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/instances", Query: "workflowID=flow&running=true"}}},
		{Name: "instance show", Command: "orchestration instances show", Args: []string{"inst"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/instances/inst", Response: `{"data":{"id":"inst","workflowID":"flow"}}`}, {Method: "GET", Path: "/workflows/flow"}}},
		{Name: "instance describe", Command: "orchestration instances describe", Args: []string{"inst"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/instances/inst/history", Response: `{"data":[{"stage":{"type":"send"}},{"stage":{"type":"delay"}}]}`}, {Method: "GET", Path: "/instances/inst/stages/0/history"}}},
		{Name: "send event", Command: "orchestration instances send-event", Args: []string{"inst", "event/a"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/instances/inst/events", Body: `{"name":"event/a"}`}}},
		{Name: "abort", Command: "orchestration instances stop", Args: []string{"inst"}, Flags: map[string]string{"confirm": "true"}, Exchanges: []testutil.Exchange{{Method: "PUT", Path: "/instances/inst/abort"}}},
		{Name: "triggers list", Command: "orchestration triggers list", Flags: map[string]string{"name": "trigger"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/triggers", Query: "name=trigger"}}},
		{Name: "trigger show", Command: "orchestration triggers show", Args: []string{"trigger"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/triggers/trigger"}}},
		{Name: "trigger create", Command: "orchestration triggers create", Args: []string{"event", "flow"}, Flags: map[string]string{"name": "trigger", "filter": "data.amount > 0", "vars": "value=data.amount"}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/triggers", Body: `{"name":"trigger","event":"event","workflowID":"flow","filter":"data.amount > 0","vars":{"value":"data.amount"}}`}}},
		{Name: "trigger test", Command: "orchestration triggers test", Args: []string{"trigger", `{"amount":9007199254740993}`}, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/v2/triggers/trigger/test", Body: `{"amount":9007199254740993}`}}},
		{Name: "trigger delete", Command: "orchestration triggers delete", Args: []string{"trigger"}, Flags: map[string]string{"confirm": "true"}, Exchanges: []testutil.Exchange{{Method: "DELETE", Path: "/triggers/trigger"}}},
		{Name: "occurrences", Command: "orchestration triggers occurrences list", Args: []string{"trigger"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/triggers/trigger/occurrences"}}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { t.Parallel(); testutil.RunCase(t, New, tc) })
	}
}
