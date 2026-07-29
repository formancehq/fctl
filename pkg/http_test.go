package fctl

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewHTTPTransportUsesProxyFromEnvironment(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool(DebugFlag, false, "")
	cmd.Flags().Bool(HTTPCloseOnErrorFlag, false, "")
	cmd.Flags().Bool(InsecureTlsFlag, false, "")

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
	if reflect.ValueOf(transport.Proxy).Pointer() != reflect.ValueOf(http.ProxyFromEnvironment).Pointer() {
		t.Fatal("expected HTTP transport proxy to be http.ProxyFromEnvironment")
	}
}
