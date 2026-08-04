package clarity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestQueryBodyRejectsNull(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("query", "", "")
	if err := cmd.Flags().Set("query", "null"); err != nil {
		t.Fatal(err)
	}

	_, err := QueryBody(cmd, "query")
	if err == nil || !strings.Contains(err.Error(), "expected an object") {
		t.Fatalf("error = %v", err)
	}
}

func TestQueryBodyWithoutValueReturnsNilInterface(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("query", "", "")
	body, err := QueryBody(cmd, "query")
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		t.Fatalf("body = %#v, want nil", body)
	}
}

func TestQueryBodyPreservesLargeNumbers(t *testing.T) {
	t.Parallel()

	const query = `{"$match":{"threshold":9007199254740993}}`
	cmd := &cobra.Command{}
	cmd.Flags().String("query", "", "")
	if err := cmd.Flags().Set("query", query); err != nil {
		t.Fatal(err)
	}

	body, err := QueryBody(cmd, "query")
	if err != nil {
		t.Fatal(err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok {
		t.Fatalf("body type = %T, want json.RawMessage", body)
	}
	if string(raw) != query {
		t.Fatalf("body = %s, want %s", raw, query)
	}
}

func TestReadJSONObjectRejectsNull(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, []byte("null"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadJSONObject(&cobra.Command{}, path)
	if err == nil || !strings.Contains(err.Error(), "expected an object") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadJSONObjectPreservesLargeNumbers(t *testing.T) {
	t.Parallel()

	const request = `{"threshold":9007199254740993}`
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, []byte(request), 0o600); err != nil {
		t.Fatal(err)
	}

	body, err := ReadJSONObject(&cobra.Command{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != request {
		t.Fatalf("body = %s, want %s", body, request)
	}
}
