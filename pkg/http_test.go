package fctl

import (
	"net/http"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewHTTPTransportUsesProxyFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example:8080")
	t.Setenv("HTTPS_PROXY", "http://secure-proxy.example:8443")
	t.Setenv("NO_PROXY", "internal.example")
	t.Setenv("REQUEST_METHOD", "")

	cmd := &cobra.Command{}

	roundTripper := NewHTTPTransport(cmd)
	headerRoundTripper, ok := roundTripper.(*injectHTTPHeadersRoundTripper)
	if !ok {
		t.Fatalf("expected header-injecting round tripper, got %T", roundTripper)
	}

	transport, ok := headerRoundTripper.next.(*http.Transport)
	if !ok {
		t.Fatalf("expected HTTP transport, got %T", headerRoundTripper.next)
	}
	if transport.Proxy == nil {
		t.Fatal("expected HTTP transport to use proxy settings from the environment")
	}

	testCases := []struct {
		name       string
		requestURL string
		wantProxy  string
	}{
		{
			name:       "HTTP proxy",
			requestURL: "http://api.example",
			wantProxy:  "http://proxy.example:8080",
		},
		{
			name:       "HTTPS proxy",
			requestURL: "https://api.example",
			wantProxy:  "http://secure-proxy.example:8443",
		},
		{
			name:       "no proxy",
			requestURL: "https://internal.example",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, testCase.requestURL, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			proxyURL, err := transport.Proxy(request)
			if err != nil {
				t.Fatalf("resolve proxy: %v", err)
			}

			gotProxy := ""
			if proxyURL != nil {
				gotProxy = proxyURL.String()
			}
			if gotProxy != testCase.wantProxy {
				t.Fatalf("expected proxy %q, got %q", testCase.wantProxy, gotProxy)
			}
		})
	}
}
