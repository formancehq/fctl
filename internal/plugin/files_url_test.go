package plugin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func pluginURLTestClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()
	client := newPluginURLClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("URL reader did not create a dedicated HTTP transport")
	}
	trusted, ok := server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatal("TLS fixture has no HTTP transport")
	}
	transport.TLSClientConfig = trusted.TLSClientConfig.Clone()
	t.Cleanup(client.CloseIdleConnections)
	return client
}

func TestPluginURLPublicTransport(t *testing.T) {
	t.Parallel()
	const contents = "chart: {}\namount: 9007199254740993\n"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/schema.yaml" || r.URL.RawQuery != "revision=legacy" {
			t.Errorf("unexpected schema request: method=%s, path=%s", r.Method, r.URL.Path)
		}
		for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
			if r.Header.Get(name) != "" {
				t.Errorf("public source request received %s", name)
			}
		}
		if _, err := io.WriteString(w, contents); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := pluginURLTestClient(t, server)
	if client.Transport == http.DefaultTransport || client.Jar != nil || client.Timeout != 30*time.Second {
		t.Fatal("URL client inherited shared transport, cookies, or an unbounded timeout")
	}
	data, err := readPluginURL(t.Context(), server.URL+"/schema.yaml?revision=legacy", client)
	if err != nil || string(data) != contents {
		t.Fatalf("HTTPS input changed: %q (%v)", data, err)
	}
}

func TestPluginURLRejectsInvalidSourceBeforeTransport(t *testing.T) {
	t.Parallel()
	for _, source := range []string{
		"http://example.test/schema", "https:opaque", "https:///schema", "https://%zz/schema",
		"https://private-user:private-password@example.test/schema", "https://@example.test/schema",
		"https://example.test/schema#private-fragment", "https://example.test/schema#",
	} {
		cmd := &cobra.Command{}
		cmd.SetContext(t.Context())
		if _, err := readPluginSource(cmd, source); err == nil {
			t.Fatal("invalid URL source accepted")
		} else if strings.Contains(err.Error(), "private-") {
			t.Fatal("URL rejection exposed credentials or a fragment")
		}
	}
}

func TestPluginURLStatusesAndResponseClosure(t *testing.T) {
	t.Parallel()
	for _, status := range []int{200, 201, 204, 206, 301, 404, 500} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			t.Parallel()
			body := &pluginURLTestBody{Reader: strings.NewReader(`{"chart":{}}`)}
			client := newPluginURLClient()
			client.Transport = pluginURLTestTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Header: http.Header{}, Body: body, Request: r}, nil
			})
			data, err := readPluginURL(t.Context(), "https://example.test/schema", client)
			accepted := status >= 200 && status < 300
			if (err == nil) != accepted || !body.closed {
				t.Fatalf("status %d: accepted=%t, closed=%t, error=%v", status, err == nil, body.closed, err)
			}
			if accepted && string(data) != `{"chart":{}}` {
				t.Fatal("successful source body changed")
			}
		})
	}
	closeErr := errors.New("response close failed")
	body := &pluginURLTestBody{Reader: strings.NewReader(`{}`), closeErr: closeErr}
	client := newPluginURLClient()
	client.Transport = pluginURLTestTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: body, Request: r}, nil
	})
	if _, err := readPluginURL(t.Context(), "https://example.test/schema", client); !errors.Is(err, closeErr) {
		t.Fatalf("response close error lost: %v", err)
	}
}

func TestPluginURLBodyLimits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		size    int
		chunked bool
	}{
		{"exact", maxPluginInput, false},
		{"oversized", maxPluginInput + 1, false},
		{"oversized chunked", maxPluginInput + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testPluginURLBodyLimit(t, tc.size, tc.chunked)
		})
	}
}

func testPluginURLBodyLimit(t *testing.T, size int, chunked bool) {
	t.Helper()
	contents := bytes.Repeat([]byte("x"), size)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if chunked {
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Error(err)
				return
			}
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(size))
		}
		if _, err := w.Write(contents); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	data, err := readPluginURL(t.Context(), server.URL, pluginURLTestClient(t, server))
	if size <= maxPluginInput {
		if err != nil || !bytes.Equal(data, contents) {
			t.Fatalf("exact-limit body rejected or changed: %v", err)
		}
	} else if err == nil || !strings.Contains(err.Error(), "4 MiB") {
		t.Fatalf("oversized response accepted: %v", err)
	}
}

func TestPluginURLRedirectRefusals(t *testing.T) {
	t.Parallel()
	for _, destination := range []string{
		"http://example.test/schema", "ftp://example.test/schema",
		"https://private-user:private-password@example.test/schema?private-query=value",
		"https://@example.test/schema", "https://example.test/schema#private-fragment", "/next#",
	} {
		var hits atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			w.Header().Set("Location", destination)
			w.WriteHeader(http.StatusFound)
		}))
		client := pluginURLTestClient(t, server)
		_, err := readPluginURL(t.Context(), server.URL+"?private-query=value", client)
		server.Close()
		if err == nil || hits.Load() != 1 {
			t.Fatalf("unsafe redirect was followed: hits=%d, error=%v", hits.Load(), err)
		}
		if strings.Contains(err.Error(), "private-") {
			t.Fatal("redirect rejection exposed credentials or URL query values")
		}
	}
}

func TestPluginURLHTTPSRedirectAndCookieIsolation(t *testing.T) {
	t.Parallel()
	const contents = `{"chart":{}}`
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("public redirect carried authentication")
		}
		if _, err := io.WriteString(w, contents); err != nil {
			t.Error(err)
		}
	}))
	defer destination.Close()
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Set-Cookie", "session=private-cookie; Secure")
		w.Header().Set("Location", destination.URL+"/schema")
		w.WriteHeader(http.StatusFound)
	}))
	defer origin.Close()
	data, err := readPluginURL(t.Context(), origin.URL, pluginURLTestClient(t, origin))
	if err != nil || string(data) != contents {
		t.Fatalf("safe HTTPS redirect failed: %q (%v)", data, err)
	}
}

func TestPluginURLRedirectLimit(t *testing.T) {
	t.Parallel()
	for _, redirects := range []int{5, 6} {
		t.Run(strconv.Itoa(redirects), func(t *testing.T) {
			t.Parallel()
			testPluginURLRedirectLimit(t, redirects)
		})
	}
}

func testPluginURLRedirectLimit(t *testing.T, redirects int) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		hop, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/hop/"))
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if hop < redirects {
			w.Header().Set("Location", "/hop/"+strconv.Itoa(hop+1))
			w.WriteHeader(http.StatusFound)
			return
		}
		if _, err := io.WriteString(w, `{"chart":{}}`); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	_, err := readPluginURL(t.Context(), server.URL+"/hop/0", pluginURLTestClient(t, server))
	if (err == nil) != (redirects == 5) || hits.Load() != 6 {
		t.Fatalf("redirect count %d: requests=%d, error=%v", redirects, hits.Load(), err)
	}
}

func TestPluginURLCancellationAndTimeout(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"headers", "body", "timeout"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			testPluginURLCancellation(t, phase)
		})
	}
}

func testPluginURLCancellation(t *testing.T, phase string) {
	t.Helper()
	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if phase != "headers" {
			w.WriteHeader(http.StatusOK)
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Error(err)
				return
			}
		}
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := pluginURLTestClient(t, server)
	want := context.Canceled
	if phase == "timeout" {
		client.Timeout = 200 * time.Millisecond
		want = context.DeadlineExceeded
	}
	finished := make(chan error, 1)
	go func() {
		_, err := readPluginURL(ctx, server.URL, client)
		finished <- err
	}()
	if phase != "timeout" {
		waitPluginURLStarted(t, started)
		cancel()
	}
	waitPluginURLError(t, finished, want)
}

func waitPluginURLStarted(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTPS request did not reach the fixture")
	}
}

func waitPluginURLError(t *testing.T, finished <-chan error, want error) {
	t.Helper()
	select {
	case err := <-finished:
		if !errors.Is(err, want) {
			t.Fatalf("HTTPS error = %v, want %v", err, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("HTTPS read did not stop")
	}
}

func TestPluginURLTransportErrorOmitsURL(t *testing.T) {
	t.Parallel()
	cause := errors.New("transport failed")
	client := newPluginURLClient()
	client.Transport = pluginURLTestTransport(func(*http.Request) (*http.Response, error) { return nil, cause })
	_, err := readPluginURL(t.Context(), "https://example.test/schema?private-token=value", client)
	if !errors.Is(err, cause) || strings.Contains(err.Error(), "private-token") {
		t.Fatalf("transport error lost its cause or exposed the URL: %v", err)
	}
}

type pluginURLTestTransport func(*http.Request) (*http.Response, error)

func (f pluginURLTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type pluginURLTestBody struct {
	*strings.Reader
	closed   bool
	closeErr error
}

func (b *pluginURLTestBody) Close() error {
	b.closed = true
	return b.closeErr
}

func TestPluginRedirectedStdinFile(t *testing.T) {
	t.Parallel()
	const contents = "send [USD/2 1] (\n  source = @world\n  destination = @users:qa:main\n)\n"
	source := filepath.Join(t.TempDir(), "transfer.num")
	if err := os.WriteFile(source, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(source) //nolint:gosec // The input is the test-owned regular file used to reproduce redirected stdin.
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := input.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	data, err := readPluginStdin(ctx, input)
	if err != nil || string(data) != contents {
		t.Fatalf("redirected stdin = %q (%v)", data, err)
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("host closed redirected stdin: %v", err)
	}
}

func TestPluginRedirectedStdinFileCanceled(t *testing.T) {
	t.Parallel()
	input, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := input.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := readPluginStdin(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled redirected stdin = %v", err)
	}
}
