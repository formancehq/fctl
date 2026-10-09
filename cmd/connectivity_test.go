package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/interactive"
)

type connectivityRunner func(context.Context, *cobra.Command, []interactive.Field) ([]string, error)

func (f connectivityRunner) Run(ctx context.Context, cmd *cobra.Command, fields []interactive.Field) ([]string, error) {
	return f(ctx, cmd, fields)
}

//nolint:gocognit // The fixture and Runner jointly verify pagination, labels and every generated creation field.
func TestConnectivityCreationFormPaginatesChoices(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	pages, writes := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/connectors" {
			pages++
			response := `{"cursor":{"pageSize":15,"hasMore":true,"next":"opaque +/=","data":[{"metadata":{"name":"adyen"},"spec":{"displayName":"Adyen"}}]}}`
			if pages == 2 {
				if r.URL.Query().Get("cursor") != "opaque +/=" {
					t.Error("choice cursor corrupted")
				}
				response = `{"cursor":{"pageSize":15,"hasMore":false,"data":[{"metadata":{"name":"stripe"},"spec":{"displayName":"Stripe"}}]}}`
			}
			writeConnectivityResponse(t, w, response)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/connectorinstances" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
			return
		}
		writes++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		var payload struct {
			Name string
			Spec struct {
				Connector, Ledger, Channel, PollInterval string
				StartSequence                            int64
				Suspend                                  bool
			}
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Error(err)
		}
		if payload.Name != "ingestion" || payload.Spec.Connector != "stripe" || payload.Spec.Ledger != "books" || payload.Spec.StartSequence != 9223372036854775807 || payload.Spec.Suspend || payload.Spec.Channel != "stable" || payload.Spec.PollInterval != "30s" {
			t.Errorf("form body=%s", body)
		}
		w.WriteHeader(http.StatusCreated)
		writeConnectivityResponse(t, w, `{"metadata":{"name":"ingestion"},"spec":{"ledger":"books"}}`)
	}))
	t.Cleanup(server.Close)
	answers := map[string]string{"Instance name": "ingestion", "Connector": "stripe", "Ledger": "books", "Channel": "stable", "Configuration": "{}", "Start sequence": "9223372036854775807", "Poll interval": "30s", "Suspend ingestion": "false"}
	runner := connectivityRunner(func(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
		values := make([]string, len(fields))
		for i, field := range fields {
			if field.Title == "Connector" && (len(field.Options) != 2 || field.Options[1].Label != "Stripe (stripe)" || field.Options[1].Value != "stripe") {
				t.Errorf("connector choices=%+v", field.Options)
			}
			values[i] = answers[field.Title]
		}
		return values, nil
	})
	root := cmd.NewRootCommand()
	root.SetArgs([]string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--connectivity-url", server.URL, "connectivity", "instances", "create"})
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(io.Discard)
	if err := root.ExecuteContext(interactive.WithRunner(t.Context(), runner)); err != nil {
		t.Fatal(err)
	}
	if pages != 2 || writes != 1 || !strings.Contains(output.String(), "ingestion") {
		t.Fatalf("pages=%d writes=%d output=%s", pages, writes, output.String())
	}
}

func TestConnectivityExplicitBodiesBypassForms(t *testing.T) {
	body := `{"suspend":false,"version":null,"startSequence":9007199254740993}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != body || r.Method != http.MethodPatch || r.Header.Get("Content-Type") != "application/merge-patch+json" {
			t.Errorf("method=%s type=%s body=%s err=%v", r.Method, r.Header.Get("Content-Type"), data, err)
		}
		writeConnectivityResponse(t, w, `{"metadata":{"name":"ingestion"}}`)
	}))
	t.Cleanup(server.Close)
	file := filepath.Join(t.TempDir(), "patch.json")
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{body, "@" + file, "-"} {
		t.Run(input[:1], func(t *testing.T) {
			root := cmd.NewRootCommand()
			root.SetArgs([]string{"--config-dir", t.TempDir(), "--no-input", "--auth-mode", "none", "--connectivity-url", server.URL, "connectivity", "instances", "patch", "ingestion", "--data", input})
			root.SetIn(strings.NewReader(body))
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			if err := root.ExecuteContext(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

//nolint:gocognit // This end-to-end fixture verifies selection and the exact merge patch produced by the form.
func TestConnectivityPatchFormAndInstanceSelection(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	patch := `{"suspend":true,"version":null}`
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/connectorinstances" {
			writeConnectivityResponse(t, w, `{"cursor":{"hasMore":false,"data":[{"metadata":{"name":"ingestion"},"spec":{"connector":"stripe","ledger":"books"},"status":{"phase":"Running"}}]}}`)
			return
		}
		writes++
		data, err := io.ReadAll(r.Body)
		if err != nil || r.Method != http.MethodPatch || r.URL.Path != "/connectorinstances/ingestion" || string(data) != patch {
			t.Errorf("patch body=%s err=%v", data, err)
		}
		writeConnectivityResponse(t, w, `{}`)
	}))
	t.Cleanup(server.Close)
	runner := connectivityRunner(func(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
		values := make([]string, len(fields))
		for i, field := range fields {
			if field.Title == "Connector instance" {
				if len(field.Options) != 1 || field.Options[0].Label != "stripe · books · Running (ingestion)" {
					t.Errorf("instance choices=%+v", field.Options)
				}
				values[i] = "ingestion"
			} else {
				values[i] = patch
			}
		}
		return values, nil
	})
	root := cmd.NewRootCommand()
	root.SetArgs([]string{"--config-dir", t.TempDir(), "--auth-mode", "none", "--connectivity-url", server.URL, "connectivity", "instances", "patch"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	if err := root.ExecuteContext(interactive.WithRunner(t.Context(), runner)); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("writes=%d", writes)
	}
}

func writeConnectivityResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(w, body); err != nil {
		t.Error(err)
	}
}
