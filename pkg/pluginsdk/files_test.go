package pluginsdk_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func TestFileSpecMetadataRoundTrip(t *testing.T) {
	t.Parallel()
	spec := pluginsdk.CommandSpec{Use: "import", Aliases: []string{"old_import"}, Files: &pluginsdk.FileSpec{
		ReadArgument: new(0), ReadFlag: "input", ReadFormat: "string",
		WriteFlag: "file", WriteFormat: "ndjson", FormatFlag: "format",
	}}
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pluginsdk.CommandSpec
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Files == nil || decoded.Files.ReadArgument == nil || *decoded.Files.ReadArgument != 0 ||
		decoded.Files.ReadFlag != "input" || decoded.Files.WriteFlag != "file" ||
		decoded.Files.ReadFormat != "string" || decoded.Files.WriteFormat != "ndjson" || decoded.Files.FormatFlag != "format" ||
		len(decoded.Aliases) != 1 || decoded.Aliases[0] != "old_import" {
		t.Fatalf("metadata changed: %s", raw)
	}
	var fields map[string]json.RawMessage
	raw, err = json.Marshal(pluginsdk.CommandSpec{Use: "list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["files"]; exists {
		t.Fatal("nil file metadata was not omitted")
	}
}
