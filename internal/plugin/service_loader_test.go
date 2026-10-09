package plugin

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

type loaderTransport func(*http.Request) (*http.Response, error)

func (f loaderTransport) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

//nolint:gocognit // The same version-cache and exact-mismatch lifecycle applies to both services.
func TestExternalServiceVersionBeforeProcess(t *testing.T) {
	for _, service := range []string{"auth", "ledger"} {
		t.Run(service, func(t *testing.T) {
			var requests atomic.Int32
			p := &external{manifest: pluginsdk.Manifest{Service: service, Version: "1.2.3"}, http: &http.Client{Transport: loaderTransport(func(request *http.Request) (*http.Response, error) {
				requests.Add(1)
				if request.Method != http.MethodGet || request.URL.Path != "/api/"+service+"/_info" {
					t.Fatalf("version request: %s %s", request.Method, request.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"version":"v1.2.3"}`))}, nil
			})}}
			endpoint := "https://stack.example/api/" + service
			for range 2 {
				if err := p.checkVersion(t.Context(), endpoint); err != nil {
					t.Fatal(err)
				}
			}
			if requests.Load() != 1 {
				t.Fatalf("version checks: %d", requests.Load())
			}
			p.manifest.Version = "1.2.4"
			p.checked = ""
			err := p.checkVersion(t.Context(), endpoint)
			if err == nil || !strings.Contains(err.Error(), service+" 1.2.4") {
				t.Fatalf("service mismatch: %v", err)
			}
		})
	}
}

func TestExternalAuthVersionFailureDoesNotStartExecutable(t *testing.T) {
	p := &external{verify: func(_ context.Context) (string, error) { return "/does/not/exist", nil }, manifest: pluginsdk.Manifest{Service: "auth", Version: "1.2.3"}, http: &http.Client{Transport: loaderTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"version":"1.2.4"}`))}, nil
	})}}
	_, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{Endpoint: "https://auth.example"})
	if err == nil || !strings.Contains(err.Error(), "targets auth 1.2.3") {
		t.Fatalf("version check did not precede startup: %v", err)
	}
}
