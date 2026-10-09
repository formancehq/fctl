package cmd_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/interactive"
)

//nolint:gocognit // Covers both form cancellation and a declined default-No destructive confirmation without HTTP writes.
func TestConnectivityCancelAndDeclineNeverWrite(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	for _, action := range []string{"create", "delete"} {
		t.Run(action, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(500) }))
			t.Cleanup(server.Close)
			runner := connectivityRunner(func(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
				if action == "create" {
					return nil, interactive.ErrCanceled
				}
				if len(fields) != 1 || fields[0].Kind != "confirm" || fields[0].Default != "false" {
					t.Error("delete confirmation must default to No")
				}
				return []string{"false"}, nil
			})
			root := cmd.NewRootCommand()
			args := []string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--connectivity-url", server.URL, "connectivity", "instances", action}
			if action == "delete" {
				args = append(args, "ingestion")
			}
			root.SetArgs(args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			if err := root.ExecuteContext(interactive.WithRunner(t.Context(), runner)); !errors.Is(err, interactive.ErrCanceled) {
				t.Fatalf("error=%v", err)
			}
			if calls != 0 {
				t.Fatalf("canceled command sent %d requests", calls)
			}
		})
	}
}
