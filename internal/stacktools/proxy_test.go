package stacktools_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/v4/internal/stacktools"
)

type authenticatedTransport struct{ base http.RoundTripper }

func (transport authenticatedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copyRequest := request.Clone(request.Context())
	copyRequest.Header.Set("Authorization", "Bearer synthetic-stack-token")
	return transport.base.RoundTrip(copyRequest)
}

func authenticatedClient(server *httptest.Server) *http.Client {
	client := *server.Client()
	client.Transport = authenticatedTransport{base: client.Transport}
	return &client
}

func TestProxyCredentialsOriginsAndPrefix(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assertProxyUpstream(t, r)
		w.Header().Set("Set-Cookie", "private=session")
		w.Header().Set("Access-Control-Allow-Origin", "https://wrong.example")
		if _, err := io.WriteString(w, "service response"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(upstream.Close)
	options := stacktools.ProxyOptions{Endpoint: upstream.URL + "/gateway", Client: authenticatedClient(upstream), Timeout: time.Second, AllowedOrigins: []string{"https://console.example"}}
	handler, err := stacktools.ProxyHandler(options)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("allowed", func(t *testing.T) { assertProxyAllowed(t, handler) })
	t.Run("denied", func(t *testing.T) {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/api/ledger/v3/ledgers", nil)
		request.Header.Set("Origin", "https://attacker.example")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("status %d", response.Code)
		}
	})
	t.Run("preflight", func(t *testing.T) {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "http://127.0.0.1/api/ledger/v3/ledgers", nil)
		request.Header.Set("Origin", "https://console.example")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || strings.Contains(response.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
			t.Fatal("invalid preflight response")
		}
	})
	if requests.Load() != 1 {
		t.Fatalf("unexpected upstream calls: %d", requests.Load())
	}
}

func assertProxyAllowed(t *testing.T, handler http.Handler) {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/api/ledger/v3/ledgers?cursor=a%2Fb", nil)
	request.Header.Set("Origin", "https://console.example")
	request.Header.Set("Authorization", "Bearer browser-secret")
	request.Header.Set("Cookie", "browser=secret")
	request.Header.Set("Referer", "https://console.example/?secret=value")
	request.Header.Set("Proxy-Authorization", "Basic browser-secret")
	request.Header.Set("X-Forwarded-For", "attacker")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "service response" {
		t.Fatalf("proxy response: %d %s", response.Code, response.Body)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "https://console.example" || response.Header().Get("Set-Cookie") != "" {
		t.Fatal("upstream CORS or cookie escaped")
	}
}

func TestProxyDoesNotFollowRedirect(t *testing.T) {
	t.Parallel()
	var external atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { external.Add(1) }))
	t.Cleanup(other.Close)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(upstream.Close)
	handler, err := stacktools.ProxyHandler(stacktools.ProxyOptions{Endpoint: upstream.URL, Client: authenticatedClient(upstream), Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/redirect", nil))
	if response.Code != http.StatusTemporaryRedirect || external.Load() != 0 {
		t.Fatal("proxy followed an upstream redirect")
	}
}

func TestProxyCancellationAndLoopback(t *testing.T) {
	t.Parallel()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	t.Cleanup(upstream.Close)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- stacktools.ServeProxy(ctx, stacktools.ProxyOptions{Endpoint: upstream.URL, Client: upstream.Client(), Port: 0, Timeout: time.Second}, func(address string) error { ready <- address; return nil })
	}()
	var address string
	select {
	case address = <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("proxy did not start")
	}
	if !strings.HasPrefix(address, "127.0.0.1:") {
		t.Fatal("proxy is not loopback")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("proxy did not stop")
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+address, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Do(request)
	if err == nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("listener remained open")
	}
}

func TestProxyTimeout(t *testing.T) {
	t.Parallel()
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	t.Cleanup(upstream.Close)
	handler, err := stacktools.ProxyHandler(stacktools.ProxyOptions{Endpoint: upstream.URL, Client: upstream.Client(), Timeout: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/slow", nil))
	if response.Code != http.StatusBadGateway {
		t.Fatalf("timeout status %d", response.Code)
	}
}

func assertProxyUpstream(t *testing.T, r *http.Request) {
	t.Helper()
	if r.URL.Path != "/gateway/api/ledger/v3/ledgers" || r.URL.RawQuery != "cursor=a%2Fb" {
		t.Errorf("unexpected upstream route: %s", r.URL)
	}
	if r.Header.Get("Authorization") != "Bearer synthetic-stack-token" {
		t.Error("stack bearer missing")
	}
	for _, name := range []string{"Cookie", "Origin", "Proxy-Authorization", "Referer", "X-Forwarded-For"} {
		if r.Header.Get(name) != "" {
			t.Errorf("upstream received %s", name)
		}
	}
}

type acquiringTransport struct {
	acquisitions *atomic.Int32
	base         http.RoundTripper
}

func (transport acquiringTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.acquisitions.Add(1)
	return authenticatedTransport{base: transport.base}.RoundTrip(request)
}

func TestProxyHostValidationBeforeAuthentication(t *testing.T) {
	t.Parallel()
	var acquisitions, requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer synthetic-stack-token" {
			t.Error("stack token missing")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)
	client := *upstream.Client()
	client.Transport = acquiringTransport{acquisitions: &acquisitions, base: client.Transport}
	handler, err := stacktools.ProxyHandler(stacktools.ProxyOptions{Endpoint: upstream.URL, Client: &client, AllowedOrigins: []string{"*"}, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	allowed := []string{"localhost", "LOCALHOST", "localhost:55001", "127.0.0.1", "127.1.2.3:55001", "[::1]", "[::1]:55001", "[::ffff:127.0.0.1]:55001"}
	proxy := httptest.NewServer(handler)
	t.Cleanup(proxy.Close)
	for _, host := range allowed {
		t.Run(host, func(t *testing.T) { assertProxyNetworkHost(t, proxy, host, http.StatusNoContent) })
	}
	before := acquisitions.Load()
	t.Run("DNS rebinding without Origin", func(t *testing.T) { assertProxyNetworkHost(t, proxy, "attacker.example:55001", http.StatusForbidden) })
	rejected := []string{"", "attacker.example", "localhost.attacker.example", "localhost.", "192.0.2.1:55001", "[2001:db8::1]:55001", "::1", "localhost:", "localhost:abc", "localhost:+80", "localhost:65536", "localhost:80:90", "[localhost]:80", "[127.0.0.1]", "[::1", "::1]", "[::1]:", "user@localhost:80", "localhost@attacker.example", "localhost/path", "localhost?query", " localhost", "localhost ", "[::1%lo0]:80"}
	for _, host := range rejected {
		t.Run("reject "+host, func(t *testing.T) { assertProxyRejectedHost(t, handler, host) })
	}
	if acquisitions.Load() != before || int(requests.Load()) != len(allowed) {
		t.Fatal("rejected Host reached token acquisition or upstream")
	}
}

func assertProxyNetworkHost(t *testing.T, proxy *httptest.Server, host string, status int) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, proxy.URL+"/api/ledger/v3/ledgers", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = host
	response, err := proxy.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Error(err)
	}
	if response.StatusCode != status {
		t.Fatalf("Host %q: status %d, want %d", host, response.StatusCode, status)
	}
}

func assertProxyRejectedHost(t *testing.T, handler http.Handler, host string) {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1/api/ledger/v3/ledgers", nil)
	request.Host = host
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("Host %q accepted with status %d", host, response.Code)
	}
	request.Header.Set("Origin", "https://attacker.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("Host %q bypassed validation through wildcard CORS", host)
	}
}
