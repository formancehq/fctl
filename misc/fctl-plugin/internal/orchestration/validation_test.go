package orchestration

import (
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func TestInvalidInputMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "empty stages", Command: "orchestration workflows create", Body: `{"stages":[]}`},
		{Name: "scalar stage", Command: "orchestration workflows create", Body: `{"stages":[1]}`},
		{Name: "null variables", Command: "orchestration workflows run", Args: []string{"flow"}, Body: `null`},
		{Name: "duplicate variables", Command: "orchestration workflows run", Args: []string{"flow"}, Flags: map[string]string{"variable": "x=1,x=2"}},
		{Name: "bad variables", Command: "orchestration triggers create", Args: []string{"event", "flow"}, Flags: map[string]string{"vars": "x"}},
		{Name: "duplicate event payload", Command: "orchestration triggers test", Args: []string{"trigger", `{}`}, Body: `{}`},
		{Name: "missing event payload", Command: "orchestration triggers test", Args: []string{"trigger"}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			testutil.RunFailure(t, New, tc, "")
		})
	}
}

func TestDependentReadValidation(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Name: "missing workflow reference", Command: "orchestration instances show", Args: []string{"id"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/instances/id", Response: `{"data":{}}`}}},
		{Name: "invalid instance history", Command: "orchestration instances describe", Args: []string{"id"}, Exchanges: []testutil.Exchange{{Method: "GET", Path: "/instances/id/history", Response: `{"data":{}}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			want := "workflowID"
			if tc.Name == "invalid instance history" {
				want = "data array"
			}
			testutil.RunFailure(t, New, tc, want)
		})
	}
}
