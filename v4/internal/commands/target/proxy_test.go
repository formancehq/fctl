package target

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewProxyHandlerForwardsRequests(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ledger" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		w.Header().Set("Access-Control-Allow-Origin", "https://upstream.example")
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	handler, err := NewProxyHandler(ProxyInput{TargetURL: upstream.URL + "/api"})
	if err != nil {
		t.Fatalf("new proxy handler: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/ledger", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected response code %d", response.Code)
	}
	if body := strings.TrimSpace(response.Body.String()); body != "ok" {
		t.Fatalf("unexpected body %q", body)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected upstream CORS header to be cleared, got %q", got)
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	handler := CORSMiddleware([]string{"https://app.example"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	request := httptest.NewRequest(http.MethodOptions, "http://proxy.test/ledger", nil)
	request.Header.Set("Origin", "https://app.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected response code %d", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("unexpected allow origin %q", got)
	}
	if got := response.Header().Get("Access-Control-Expose-Headers"); got != "Count" {
		t.Fatalf("unexpected exposed headers %q", got)
	}
}
