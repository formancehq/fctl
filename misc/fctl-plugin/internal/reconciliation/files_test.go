package reconciliation

import (
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

// The host decodes historical file inputs and leaves their source argument in place.
func TestHistoricalFileArguments(t *testing.T) {
	t.Parallel()
	confirm := map[string]string{"confirm": "true"}
	cases := []testutil.Case{
		{Command: "reconciliation policies create", Flags: confirm, Body: `{"name":"qa","ledgerName":"qa","paymentsPoolID":"pool","ledgerQuery":{}}`, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/policies", Body: `{"name":"qa","ledgerName":"qa","paymentsPoolID":"pool","ledgerQuery":{}}`}}},
		{Command: "reconciliation rules create", Flags: confirm, Body: `{"name":"qa","templateKind":"ledger_invariant","templateSpec":{"terms":[],"tolerance":{"USD/2":0}}}`, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/rules", Body: `{"name":"qa","templateKind":"ledger_invariant","templateSpec":{"terms":[],"tolerance":{"USD/2":0}}}`}}},
		{Command: "reconciliation rules update", Args: []string{"rule"}, Flags: confirm, Body: `{"enabled":false}`, Exchanges: []testutil.Exchange{{Method: "PATCH", Path: "/rules/rule", Body: `{"enabled":false}`}}},
	}
	for _, tc := range cases {
		t.Run(tc.Command, func(t *testing.T) {
			t.Parallel()
			path := strings.Fields(tc.Command)
			manifest, err := New(nil).GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			command, err := pluginsdk.FindCommand(manifest, path)
			if err != nil {
				t.Fatal(err)
			}
			if command.Files == nil || command.Files.ReadArgument == nil || *command.Files.ReadArgument != len(tc.Args) || command.Files.ReadFormat != "yaml" {
				t.Fatalf("missing historical file input: %+v", command.Files)
			}
			if command.Args.Min != len(tc.Args) || command.Args.Max != len(tc.Args)+1 || !strings.Contains(command.Use, "[<file>|-]") {
				t.Fatalf("wrong file signature: %+v", command)
			}
			tc.Args = append(append([]string{}, tc.Args...), "/does-not-exist/fixture.yaml")
			testutil.RunCase(t, New, tc)
		})
	}
}
