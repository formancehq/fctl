package wallets

import (
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func TestFileInputsOnlyOnHistoricalPayloadCommands(t *testing.T) {
	t.Parallel()
	cases := []string{"wallets create"}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			names := strings.Fields(path)
			manifest, err := New(nil).GetManifest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			command, err := pluginsdk.FindCommand(manifest, names)
			if err != nil {
				t.Fatal(err)
			}
			if command.Files != nil {
				t.Fatal("non-file historical command gained a file source")
			}
		})
	}
}
