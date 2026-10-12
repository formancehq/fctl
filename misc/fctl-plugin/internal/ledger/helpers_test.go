package ledger

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

const huge = "1234567890123456789012345678901234567890"

func executeRequest(command string, args []string, flags map[string]string, body string) pluginsdk.ExecuteRequest {
	values := make(map[string]string)
	maps.Copy(values, flags)
	path := append([]string{"ledger"}, strings.Fields(command)...)
	if spec, err := pluginsdk.FindCommand(manifest(), path); err == nil && spec.Confirm {
		if _, exists := values["confirm"]; !exists {
			values["confirm"] = "true"
		}
	}
	r := pluginsdk.ExecuteRequest{CommandPath: path, Args: args, Flags: values}
	if body != "" {
		r.Body = json.RawMessage(body)
	}
	return r
}

func writeJSON(t *testing.T, w http.ResponseWriter, data string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.WriteString(w, data); err != nil {
		t.Error(err)
	}
}

func assertJSON(t *testing.T, got, want []byte) {
	t.Helper()
	decode := func(raw []byte) any {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("invalid JSON %s: %v", raw, err)
		}
		return value
	}
	if !reflect.DeepEqual(decode(got), decode(want)) {
		t.Errorf("JSON %s, want %s", got, want)
	}
}
