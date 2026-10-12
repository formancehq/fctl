package orchestration

import (
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

// The host decodes historical file inputs and leaves their source argument in place.
func TestHistoricalFileArguments(t *testing.T) {
	t.Parallel()
	cases := []testutil.Case{
		{Command: "orchestration workflows create", Body: `{"stages":[{"delay":{"duration":"1s"}}]}`, Exchanges: []testutil.Exchange{{Method: "POST", Path: "/workflows", Body: `{"stages":[{"delay":{"duration":"1s"}}]}`}}},
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
