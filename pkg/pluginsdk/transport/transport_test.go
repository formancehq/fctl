package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

// The test binary is a real go-plugin executable when launched by Open. It
// deliberately has no test-selection argument or inherited helper variable.
func TestMain(m *testing.M) {
	if os.Getenv(handshake.MagicCookieKey) == handshake.MagicCookieValue {
		Serve(func(client *http.Client) pluginsdk.Plugin { return &helperPlugin{http: client} })
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type helperPlugin struct{ http *http.Client }

func helperManifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "test", Version: "3.0.0", Service: "ledger", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: "test", Runnable: true, Args: pluginsdk.ArgsSpec{Max: 1},
			Flags: []pluginsdk.FlagSpec{
				{Name: "action", Type: "string", Default: "echo"}, {Name: "path", Type: "string", Default: "/v3/books"},
				{Name: "method", Type: "string", Default: "GET"}, {Name: "url", Type: "string"},
				{Name: "header", Type: "string"}, {Name: "partial", Type: "bool", Default: "false"},
				{Name: "size", Type: "uint32", Default: "0"}, {Name: "enabled", Type: "bool", Default: "true"},
			}, Inputs: []pluginsdk.InputSpec{{Title: "Book", Kind: "select", Flag: "path", Required: true,
				Source: &pluginsdk.ChoiceSource{CommandPath: []string{"test"}, Flags: map[string]string{"action": "http"},
					ValueField: "name", MatchFields: map[string][]string{"state": {"ready"}}, ExcludeTrueFields: []string{"disabled"}}}},
		}}
}

func (h *helperPlugin) GetManifest(context.Context) (pluginsdk.Manifest, error) {
	if h.http != nil {
		return pluginsdk.Manifest{}, fmt.Errorf("manifest inspection received an HTTP client")
	}
	return helperManifest(), nil
}

func (h *helperPlugin) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	request, err := pluginsdk.NormalizeRequest(helperManifest(), request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	switch request.Flags["action"] {
	case "environment":
		data, err := json.Marshal(struct {
			Environment []string `json:"environment"`
			Arguments   []string `json:"arguments"`
		}{os.Environ(), os.Args})
		return pluginsdk.ExecuteResponse{Data: data}, err
	case "request":
		data, err := json.Marshal(toWire(request))
		return pluginsdk.ExecuteResponse{Data: data}, err
	case "error":
		data := []byte(` {"id":90071992547409930001,"results":[{"status":"ok"}]} `)
		return pluginsdk.ExecuteResponse{Data: data}, fmt.Errorf("wrapped: %w", &httpclient.Error{StatusCode: 409,
			Code: "PARTIAL", Message: "one operation failed", Body: []byte(` {"failed":18446744073709551615000} `)})
	case "large":
		size, err := strconv.Atoi(request.Flags["size"])
		if err != nil {
			return pluginsdk.ExecuteResponse{}, err
		}
		return pluginsdk.ExecuteResponse{Data: []byte(`"` + strings.Repeat("x", size) + `"`)}, nil
	case "ignore-context":
		select {}
	case "cancel":
		<-ctx.Done()
		return pluginsdk.ExecuteResponse{}, ctx.Err()
	case "crash":
		os.Exit(23)
	case "raw-http":
		return h.rawHTTP(ctx, request)
	case "http":
		client, err := httpclient.New(request.Endpoint, h.http)
		if err != nil {
			return pluginsdk.ExecuteResponse{}, err
		}
		headers := http.Header{}
		if name := request.Flags["header"]; name != "" {
			headers.Set(name, "plugin-controlled")
		}
		data, err := client.Do(ctx, request.Flags["method"], request.Flags["path"], nil, request.Body, headers)
		if failure, ok := errors.AsType[*httpclient.Error](err); ok && request.Flags["partial"] == "true" {
			data = failure.Body
		}
		return pluginsdk.ExecuteResponse{Data: data}, err
	}
	return pluginsdk.ExecuteResponse{Data: request.Body}, nil
}

func (h *helperPlugin) rawHTTP(ctx context.Context, request pluginsdk.ExecuteRequest) (result pluginsdk.ExecuteResponse, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", request.Flags["url"], nil)
	if err != nil {
		return result, err
	}
	response, err := h.http.Do(req)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	data, err := json.Marshal(response.Header)
	return pluginsdk.ExecuteResponse{Data: data}, err
}

func helperBinary(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func openHelper(t *testing.T, client *http.Client, endpoint string) *Client {
	t.Helper()
	runtime, err := Open(t.Context(), helperBinary(t), client, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
		if !runtime.process.Exited() {
			t.Error("Close did not reap plugin process")
		}
	})
	return runtime
}

func helperRequest(action string) pluginsdk.ExecuteRequest {
	return pluginsdk.ExecuteRequest{CommandPath: []string{"test"}, Flags: map[string]string{"action": action}}
}

func TestManifestWorksWithoutHTTPAndPreservesForms(t *testing.T) {
	t.Parallel()
	client := openHelper(t, nil, "")
	manifest, err := client.GetManifest(t.Context())
	if err != nil || !reflect.DeepEqual(manifest, helperManifest()) {
		t.Fatalf("manifest round trip = %+v, %v", manifest, err)
	}
	if _, err := client.Execute(t.Context(), helperRequest("http")); err == nil {
		t.Fatal("execution accepted missing host HTTP client")
	}
}

func TestRoundTripPreservesExactJSONAndRequestMetadata(t *testing.T) {
	t.Parallel()
	client := openHelper(t, http.DefaultClient, "https://unused.example/api/ledger")
	body := []byte(" {\n\"amount\":90071992547409930001, \"negative\":-18446744073709551615000} \n")
	request := helperRequest("echo")
	request.Body = body
	response, err := client.Execute(t.Context(), request)
	if err != nil || !bytes.Equal(response.Data, body) {
		t.Fatalf("exact JSON changed: %s, %v", response.Data, err)
	}
	request.Flags["action"] = "request"
	request.Args = []string{"large-ID-18446744073709551615000"}
	request.Flags["enabled"] = "false"
	request.ChangedFlags = map[string]bool{"enabled": true, "partial": false}
	request.Context = map[string]string{"organization": "org", "stack": "stack"}
	response, err = client.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	var received requestWire
	if err := json.Unmarshal(response.Data, &received); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(received.Body, body) || !reflect.DeepEqual(received.Args, request.Args) ||
		!reflect.DeepEqual(received.Context, request.Context) || !reflect.DeepEqual(received.ChangedFlags, request.ChangedFlags) ||
		received.Flags["enabled"] != "false" || received.Endpoint != "https://unused.example/api/ledger" {
		t.Fatalf("request metadata lost: %+v", received)
	}
}

func TestStructuredErrorAndPartialData(t *testing.T) {
	t.Parallel()
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	response, err := client.Execute(t.Context(), helperRequest("error"))
	failure, ok := errors.AsType[*httpclient.Error](err)
	if !ok || failure.StatusCode != 409 || failure.Code != "PARTIAL" || failure.Message != "one operation failed" ||
		string(failure.Body) != ` {"failed":18446744073709551615000} ` || string(response.Data) != ` {"id":90071992547409930001,"results":[{"status":"ok"}]} ` {
		t.Fatalf("structured error or partial data changed: %s, %+v", response.Data, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (r roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return r(request) }

func TestHTTPRemainsHostOwnedAndDoesNotExposeCredentials(t *testing.T) {
	t.Setenv("FCTL_TEST_SECRET", "never-inherit-this-secret")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "never-inherit-this-secret")
	t.Setenv("FCTL_ACCESS_TOKEN", "never-inherit-this-secret")
	var attempts atomic.Int32
	const body = " {\n\"amount\":90071992547409930001} \n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer host-only-token" {
			t.Error("host transport did not supply authentication")
		}
		if r.URL.EscapedPath() != "/api/ledger/v3/books/users%2F001" || r.Method != "POST" {
			t.Errorf("routing changed: %s %s", r.Method, r.URL.EscapedPath())
		}
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != body {
			t.Errorf("HTTP body changed: %q, %v", data, err)
		}
		w.Header().Set("Set-Cookie", "session=host-only-token")
		w.Header().Set("Authorization", "Bearer host-only-token")
		writeResponse(t, w, body)
	}))
	defer server.Close()
	host := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		attempts.Add(1) // The host's debug/auth/refresh transport remains on this path.
		request = request.Clone(request.Context())
		request.Header.Set("Authorization", "Bearer host-only-token")
		return http.DefaultTransport.RoundTrip(request)
	})}
	client := openHelper(t, host, server.URL+"/api/ledger")
	request := helperRequest("environment")
	response, err := client.Execute(t.Context(), request)
	if err != nil || bytes.Contains(response.Data, []byte("never-inherit-this-secret")) {
		t.Fatalf("secret inherited: %v", err)
	}
	assertPrivateEnvironment(t, response.Data)
	request = helperRequest("http")
	request.Flags["path"] = "/v3/books/users%2F001"
	request.Flags["method"] = "POST"
	request.Body = []byte(body)
	response, err = client.Execute(t.Context(), request)
	if err != nil || string(response.Data) != body || attempts.Load() != 1 {
		t.Fatalf("host callback = %q, %v, attempts=%d", response.Data, err, attempts.Load())
	}
}

func TestHTTPFailurePreservesStructuredPartialResponseWithoutRetry(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	const body = ` {"errorCode":"PARTIAL","errorMessage":"one failed","results":[{"id":90071992547409930001}]} `
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusConflict)
		writeResponse(t, w, body)
	}))
	defer server.Close()
	client := openHelper(t, server.Client(), server.URL+"/api/ledger")
	request := helperRequest("http")
	request.Flags["partial"] = "true"
	request.Flags["method"] = "POST"
	response, err := client.Execute(t.Context(), request)
	failure, ok := errors.AsType[*httpclient.Error](err)
	if !ok || failure.Code != "PARTIAL" || failure.StatusCode != 409 || string(failure.Body) != body || string(response.Data) != body || calls.Load() != 1 {
		t.Fatalf("HTTP failure changed or retried: data=%s, err=%v, calls=%d", response.Data, err, calls.Load())
	}
}

func TestHTTPResponsesHideCredentialHeaders(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Authorization", "Bearer hidden")
		w.Header().Set("Set-Cookie", "hidden")
		w.Header().Set("X-Request-Id", "visible")
		writeResponse(t, w, `{}`)
	}))
	defer server.Close()
	client := openHelper(t, server.Client(), server.URL+"/api/ledger")
	request := helperRequest("raw-http")
	request.Flags["url"] = server.URL + "/api/ledger/v3/books"
	response, err := client.Execute(t.Context(), request)
	if err != nil || bytes.Contains(response.Data, []byte("hidden")) || !bytes.Contains(response.Data, []byte("visible")) {
		t.Fatalf("credential headers escaped: %s, %v", response.Data, err)
	}
}

func TestHTTPRejectsEndpointEscapesAndCredentialHeaders(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeResponse(t, w, `{}`)
	}))
	defer server.Close()
	client := openHelper(t, server.Client(), server.URL+"/api/ledger")
	for _, suffix := range []string{
		"/api/auth/v3", "/api/ledger-other/v3", "/api/ledger/../auth", "/api/ledger/%2e%2e/auth",
		"/api/ledger/%252e%252e/auth", "/api/ledger/v3%2f..%2f..%2fauth", "/api/ledger/%5c..%5cauth",
	} {
		t.Run(suffix, func(t *testing.T) {
			request := helperRequest("raw-http")
			request.Flags["url"] = server.URL + suffix
			if _, err := client.Execute(t.Context(), request); err == nil {
				t.Fatal("HTTP callback allowed endpoint escape")
			}
		})
	}
	for _, target := range []string{"http://127.0.0.1:1/api/ledger/v3", strings.Replace(server.URL, "http:", "https:", 1) + "/api/ledger/v3", "http://user:password@" + strings.TrimPrefix(server.URL, "http://") + "/api/ledger/v3"} {
		request := helperRequest("raw-http")
		request.Flags["url"] = target
		if _, err := client.Execute(t.Context(), request); err == nil {
			t.Fatal("HTTP callback allowed changed origin or credentials")
		}
	}
	for _, name := range []string{"Authorization", "Cookie", "Proxy-Authorization", "X-Api-Key", "Host"} {
		request := helperRequest("http")
		request.Flags["header"] = name
		if _, err := client.Execute(t.Context(), request); err == nil {
			t.Fatalf("HTTP callback accepted %s", name)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("rejected request reached the endpoint %d times", calls.Load())
	}
}

func TestHTTPChecksRedirectBoundaries(t *testing.T) {
	t.Parallel()
	var escaped atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		escaped.Add(1)
		writeResponse(t, w, `{}`)
	}))
	defer outside.Close()
	var server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ledger/inside":
			http.Redirect(w, r, "/api/ledger/ok", http.StatusFound)
		case "/api/ledger/origin":
			http.Redirect(w, r, outside.URL, http.StatusFound)
		case "/api/ledger/sibling":
			http.Redirect(w, r, "/api/auth/secret", http.StatusFound)
		case "/api/ledger/ok":
			writeResponse(t, w, `{"ok":true}`)
		default:
			escaped.Add(1)
			writeResponse(t, w, `{}`)
		}
	}))
	defer server.Close()
	client := openHelper(t, server.Client(), server.URL+"/api/ledger")
	for _, path := range []string{"/origin", "/sibling"} {
		request := helperRequest("http")
		request.Flags["path"] = path
		if _, err := client.Execute(t.Context(), request); err == nil {
			t.Fatal("host followed redirect outside endpoint")
		}
	}
	request := helperRequest("http")
	request.Flags["path"] = "/inside"
	response, err := client.Execute(t.Context(), request)
	if err != nil || string(response.Data) != `{"ok":true}` || escaped.Load() != 0 {
		t.Fatalf("redirect behavior = %s, %v; escapes=%d", response.Data, err, escaped.Load())
	}
}

func TestTransportErrorsDoNotExposeHostSecrets(t *testing.T) {
	t.Parallel()
	host := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("private transport: Bearer host-secret at /private/auth/credentials")
	})}
	client := openHelper(t, host, "https://unused.example")
	_, err := client.Execute(t.Context(), helperRequest("http"))
	if err == nil || strings.Contains(err.Error(), "host-secret") || strings.Contains(err.Error(), "/private/auth") {
		t.Fatalf("host transport disclosed credentials: %v", err)
	}
}

func TestCancelRPCStopsUncooperativePlugin(t *testing.T) {
	t.Parallel()
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := client.Execute(ctx, helperRequest("ignore-context"))
	if !errors.Is(err, context.DeadlineExceeded) || !client.process.Exited() {
		t.Fatalf("deadline did not stop plugin: %v, exited=%v", err, client.process.Exited())
	}
	if _, err := client.GetManifest(t.Context()); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed runtime restarted: %v", err)
	}
}

func TestCancellationPropagatesToHostHTTP(t *testing.T) {
	t.Parallel()
	started, stopped := make(chan struct{}), make(chan struct{})
	host := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		close(started)
		<-request.Context().Done()
		close(stopped)
		return nil, request.Context().Err()
	})}
	client := openHelper(t, host, "https://unused.example")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := client.Execute(ctx, helperRequest("http")); finished <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP callback did not start")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled execution did not finish")
	}
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("host HTTP callback did not stop")
	}
}

func TestOpenLifetimeCancellationAndConcurrentClose(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	client, err := Open(ctx, helperBinary(t), nil, "")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	var calls sync.WaitGroup
	for range 8 {
		calls.Go(func() {
			if err := client.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	calls.Wait()
	if !client.process.Exited() {
		t.Fatal("canceled Open context retained the process")
	}
}

func TestConcurrentExecutionsUseIndependentHTTPCallbacks(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, fmt.Sprintf(`{"path":%q}`, r.URL.Path))
	}))
	defer server.Close()
	client := openHelper(t, server.Client(), server.URL+"/api/ledger")
	var calls sync.WaitGroup
	for i := range 5 {
		calls.Go(func() {
			request := helperRequest("http")
			request.Flags["path"] = "/" + strconv.Itoa(i)
			response, err := client.Execute(t.Context(), request)
			if err != nil || !strings.Contains(string(response.Data), "/api/ledger/"+strconv.Itoa(i)) {
				t.Errorf("callback crossed invocations: %s, %v", response.Data, err)
			}
		})
	}
	calls.Wait()
}

func TestPluginCrashReturnsErrorAndCanBeClosed(t *testing.T) {
	t.Parallel()
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	if _, err := client.Execute(t.Context(), helperRequest("crash")); err == nil {
		t.Fatal("plugin crash did not report error")
	}
}

func TestInputValidationAndTransportLimits(t *testing.T) {
	t.Parallel()
	for _, endpoint := range []string{"ftp://host", "https://user:secret@host", "https://host/path?token=secret", "https://host/../path"} {
		if _, err := Open(t.Context(), helperBinary(t), nil, endpoint); err == nil {
			t.Fatalf("accepted invalid endpoint %q", endpoint)
		}
	}
	if _, err := Open(t.Context(), "relative-plugin", nil, ""); err == nil {
		t.Fatal("accepted relative binary path")
	}
	if _, err := Open(t.Context(), "/missing/fctl-plugin", nil, ""); err == nil {
		t.Fatal("missing binary did not fail")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Open(ctx, helperBinary(t), nil, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled Open = %v", err)
	}
	client := openHelper(t, http.DefaultClient, "https://unused.example")
	for _, mutate := range []func(*pluginsdk.ExecuteRequest){
		func(r *pluginsdk.ExecuteRequest) { r.Endpoint = "https://other.example" },
		func(r *pluginsdk.ExecuteRequest) { r.Body = bytes.Repeat([]byte("x"), maxBodyBytes+1) },
		func(r *pluginsdk.ExecuteRequest) { r.Context = map[string]string{"client_secret": "secret"} },
		func(r *pluginsdk.ExecuteRequest) {
			r.Context = map[string]string{"note": strings.Repeat("x", maxMetaBytes)}
		},
	} {
		request := helperRequest("echo")
		mutate(&request)
		if _, err := client.Execute(t.Context(), request); err == nil {
			t.Fatal("invalid execution passed validation")
		}
	}
	request := helperRequest("large")
	request.Flags["size"] = strconv.Itoa(maxDataBytes)
	if _, err := client.Execute(t.Context(), request); err == nil {
		t.Fatal("accepted plugin result exceeding 32 MiB")
	}
}

func TestHTTPBodyLimits(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.Copy(w, io.LimitReader(&repeatingReader{}, maxDataBytes+1)); err != nil && !strings.Contains(err.Error(), "broken pipe") && !strings.Contains(err.Error(), "connection reset") {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := openHelper(t, server.Client(), server.URL)
	if _, err := client.Execute(t.Context(), helperRequest("http")); err == nil {
		t.Fatal("accepted HTTP response exceeding 32 MiB")
	}
}

type repeatingReader struct{}

func (*repeatingReader) Read(buffer []byte) (int, error) {
	for i := range buffer {
		buffer[i] = 'x'
	}
	return len(buffer), nil
}

func TestEndpointAllowsOrdinaryEncodedIdentifiers(t *testing.T) {
	t.Parallel()
	base := mustURL(t, "https://stack.example/api/ledger")
	for _, path := range []string{"/api/ledger", "/api/ledger/v3/books/a%2Fb", "/api/ledger/v3/books/100%25", "/api/ledger/v3/books/x%252Fy"} {
		target := mustURL(t, "https://stack.example"+path)
		if err := validateEndpoint(base, target); err != nil {
			t.Fatalf("encoded identifier %s rejected: %v", path, err)
		}
	}
}

func assertPrivateEnvironment(t *testing.T, data []byte) {
	t.Helper()
	var environment struct {
		Environment []string `json:"environment"`
		Arguments   []string `json:"arguments"`
	}
	if err := json.Unmarshal(data, &environment); err != nil || len(environment.Arguments) != 1 {
		t.Fatalf("plugin received process arguments: %+v, %v", environment, err)
	}
	for _, value := range environment.Environment {
		if strings.HasPrefix(value, "AWS_") || strings.HasPrefix(value, "FCTL_ACCESS_TOKEN=") || strings.HasPrefix(value, "FCTL_TEST_SECRET=") {
			t.Fatal("plugin inherited host credentials")
		}
	}
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	u, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func writeResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	//nolint:gosec // JSON test fixtures include deliberately oversized and malformed bodies.
	if _, err := io.WriteString(w, body); err != nil {
		t.Error(err)
	}
}
