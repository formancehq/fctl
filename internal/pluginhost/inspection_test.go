package pluginhost

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/formancehq/fctl/v4/internal/pluginselection"
)

func TestInspectionSelectionSeparatesGatewayAndStandaloneEndpoints(t *testing.T) {
	t.Setenv("FCTL_PROFILE", "")
	for _, service := range []string{"ledger", "auth"} {
		t.Run(service, func(t *testing.T) {
			testInspectionSelectionEndpoints(t, service)
		})
	}
}

func testInspectionSelectionEndpoints(t *testing.T, service string) {
	t.Helper()
	root, settings := loaderSettings(t)
	const endpoint = "https://stack.example/gateway"
	loaderFlag(t, root, "stack-url", endpoint+"///")
	gateway, err := SelectionTarget(settings, root, service)
	if err != nil {
		t.Fatal(err)
	}
	client, err := settings.Client(t.Context(), root, service)
	if err != nil || gateway.Endpoint != endpoint+"/api/"+service || gateway.Endpoint != client.Endpoint() {
		t.Fatalf("selection does not identify the actual gateway service: target=%+v err=%v", gateway, err)
	}
	directory := filepath.Join(settings.Directory, "plugins")
	selected := pluginselection.Selection{Target: gateway, Service: service, ServiceVersion: "2.4.15", Provider: "legacy", PluginVersion: "1.0.0"}
	if err := pluginselection.Save(directory, selected); err != nil {
		t.Fatal(err)
	}
	// The same literal URL has a different meaning as a service override.
	loaderFlag(t, root, service+"-url", endpoint)
	standalone, err := SelectionTarget(settings, root, service)
	if err != nil || standalone.Endpoint != endpoint || standalone == gateway {
		t.Fatalf("standalone target collided with gateway: target=%+v err=%v", standalone, err)
	}
	client, err = settings.Client(t.Context(), root, service)
	if err != nil || standalone.Endpoint != client.Endpoint() {
		t.Fatalf("direct selection differs from request endpoint: target=%+v err=%v", standalone, err)
	}
	if _, err := pluginselection.Load(directory, standalone, service); !errors.Is(err, pluginselection.ErrNotSelected) {
		t.Fatalf("standalone connection reused gateway selection: %v", err)
	}
	if cached, err := pluginselection.Load(directory, gateway, service); err != nil || cached != selected {
		t.Fatalf("gateway selection changed: %+v err=%v", cached, err)
	}
}

func TestInspectionSnapshotUsesExactRequestEndpoint(t *testing.T) {
	t.Setenv("FCTL_PROFILE", "")
	for _, service := range []string{"ledger", "auth"} {
		t.Run(service, func(t *testing.T) { testInspectionSnapshotEndpoint(t, service) })
	}
}

func testInspectionSnapshotEndpoint(t *testing.T, service string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("inspection made a mutation: %s %s", r.Method, r.URL.Path)
		}
		version := "2.4.15"
		if r.URL.Path != "/gateway/api/"+service+"/_info" {
			t.Errorf("inspection queried the wrong endpoint: %s", r.URL.Path)
			version = "3.0.0"
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"version": version}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "stack-url", server.URL+"/gateway/")
	selection, err := InspectService(t.Context(), root, settings, service)
	if err != nil {
		t.Fatal(err)
	}
	want := pluginselection.Selection{Target: pluginselection.Target{Endpoint: server.URL + "/gateway/api/" + service}, Service: service, ServiceVersion: "2.4.15"}
	if selection != want {
		t.Fatalf("inspected snapshot = %+v, want %+v", selection, want)
	}
}
