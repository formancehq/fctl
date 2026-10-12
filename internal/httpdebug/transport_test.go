package httpdebug_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/formancehq/fctl/v4/internal/httpdebug"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func debugRequest(t *testing.T, method, endpoint, body, contentType string) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), method, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", contentType)
	return r
}

func fixtureResponse(body, contentType string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body))}
}

func consumeResponse(t *testing.T, response *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertHidden(t *testing.T, trace string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if strings.Contains(trace, secret) {
			t.Errorf("trace leaked %q: %s", secret, trace)
		}
	}
}

func TestDeploySecretsAreRedactedWithoutChangingResponse(t *testing.T) {
	body := `{"name":"visible-variable","variables":[{"key":"credential","value":"secret-variable"}],"terraformState":{"outputs":{"password":"secret-state"}},"tfstate":"secret-raw-state"}`
	var trace bytes.Buffer
	req := debugRequest(t, "POST", "https://example.test/variables", body, "application/json")
	response, err := httpdebug.New(roundTripFunc(func(got *http.Request) (*http.Response, error) {
		data, readErr := io.ReadAll(got.Body)
		if readErr != nil || string(data) != body {
			t.Fatalf("request body changed: %s %v", data, readErr)
		}
		if closeErr := got.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		return fixtureResponse(body, "application/json"), nil
	}), &trace).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := consumeResponse(t, response); got != body {
		t.Fatal("response body changed")
	}
	assertHidden(t, trace.String(), "secret-variable", "secret-state", "secret-raw-state")
	if !strings.Contains(trace.String(), "visible-variable") {
		t.Fatal("trace lost non-secret variable name")
	}
}

func TestTraceRedactsNestedJSONHeadersAndURLsWithoutMutation(t *testing.T) {
	requestBody := `{"client_secret":"secret-client","nested":[{"access_token":"secret-access","password":"secret-password","assertion":"secret-assertion"}],"name":"visible","amount":9007199254740993}`
	responseBody := `{"id_token":"secret-id","refresh_token":"secret-refresh","error_description":"unstructured echoed credential","verification_uri_complete":"https://example.test/verify?user_code=secret-verification&locale=en","data":{"name":"response-visible"}}`
	req := debugRequest(t, "POST", "https://username:secret-url@example.test/oauth?device_code=secret-device&user_code=secret-user&client_secret=secret-query&locale=en#secret-fragment", requestBody, "application/json")
	req.Header.Set("Authorization", "Bearer secret-header")
	req.Header.Set("Cookie", "sid=secret-cookie")
	originalHeaders, originalURL := req.Header.Clone(), req.URL.String()
	var trace bytes.Buffer
	base := roundTripFunc(func(got *http.Request) (*http.Response, error) {
		if got != req || got.Context() != t.Context() {
			t.Fatal("transport replaced request or context")
		}
		data, err := io.ReadAll(got.Body)
		if err != nil || string(data) != requestBody {
			t.Fatalf("request body changed: %s %v", data, err)
		}
		if err := got.Body.Close(); err != nil {
			t.Fatal(err)
		}
		response := fixtureResponse(responseBody, "application/json")
		response.Header.Set("Set-Cookie", "sid=secret-set-cookie")
		response.Header.Set("Location", "https://example.test/verify?user_code=secret-location")
		return response, nil
	})
	response, err := httpdebug.New(base, &trace).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := consumeResponse(t, response); got != responseBody {
		t.Fatal("response body changed")
	}
	assertHidden(t, trace.String(), "secret-", "username", "unstructured echoed credential")
	for _, visible := range []string{"POST", "response 200", "visible", "9007199254740993", "locale=en", "[REDACTED]"} {
		if !strings.Contains(trace.String(), visible) {
			t.Errorf("trace missing %q: %s", visible, &trace)
		}
	}
	if req.URL.String() != originalURL || !reflect.DeepEqual(req.Header, originalHeaders) {
		t.Fatal("redaction modified outgoing URL or headers")
	}
}

func TestFormTracingRedactsOAuthFieldsAndErrorText(t *testing.T) {
	body := "client_id=fctl&client_secret=secret-form&password=secret-password&device_code=secret-device&user_code=secret-user&id_token_hint=secret-hint&assertion=secret-assertion&token=secret-token&error_description=echoed-secret&scope=openid"
	req := debugRequest(t, "POST", "https://example.test/token", body, "application/x-www-form-urlencoded; charset=utf-8")
	var trace bytes.Buffer
	response, err := httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(body, "application/x-www-form-urlencoded"), nil
	}), &trace).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := consumeResponse(t, response); got != body {
		t.Fatal("form response changed")
	}
	assertHidden(t, trace.String(), "secret-", "echoed-secret")
	if !strings.Contains(trace.String(), "scope=openid") || !strings.Contains(trace.String(), "client_id=fctl") {
		t.Fatal("non-sensitive form fields missing")
	}
}

func TestAuthSecretClearIsRedactedInNestedJSON(t *testing.T) {
	body := `{"data":{"id":"s1","clear":"secret-created","nested":[{"clear":"secret-nested","lastDigits":"1234"}]},"clear":"secret-top-level"}`
	var trace bytes.Buffer
	response, err := httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(body, "application/json"), nil
	}), &trace).RoundTrip(debugRequest(t, "POST", "https://example.test/clients/c1/secrets", body, "application/json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := consumeResponse(t, response); got != body {
		t.Fatal("redaction changed the returned Auth secret")
	}
	assertHidden(t, trace.String(), "secret-created", "secret-nested", "secret-top-level")
	if strings.Count(trace.String(), `"clear":"[REDACTED]"`) != 6 || !strings.Contains(trace.String(), `"lastDigits":"1234"`) {
		t.Fatal("request and response did not redact every clear field or lost safe metadata")
	}
}

func TestUnknownMalformedAndOversizedBodiesAreNeverPrinted(t *testing.T) {
	for _, tc := range []struct{ name, kind, body string }{
		{"unknown", "", "secret-raw"},
		{"text", "text/plain", "secret-text"},
		{"binary", "application/octet-stream", "secret-binary"},
		{"malformed json", "application/json", `{"secret":"secret-malformed"`},
		{"malformed form", "application/x-www-form-urlencoded", "name=secret-form%ZZ"},
		{"oversized json", "application/json", `{"visible":"secret-large` + strings.Repeat("x", 128<<10) + `"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var trace bytes.Buffer
			req := debugRequest(t, "POST", "https://example.test", tc.body, tc.kind)
			response, err := httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
				return fixtureResponse(tc.body, tc.kind), nil
			}), &trace).RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			if got := consumeResponse(t, response); got != tc.body {
				t.Fatal("omitted response was truncated or changed")
			}
			assertHidden(t, trace.String(), "secret-")
			if !strings.Contains(trace.String(), "omitted") || trace.Len() > 4096 {
				t.Fatal("body trace was not bounded and summarized")
			}
		})
	}
}

type trackedBody struct {
	reader io.Reader
	reads  int
	closed bool
}

func (b *trackedBody) Read(p []byte) (int, error) { b.reads++; return b.reader.Read(p) }
func (b *trackedBody) Close() error               { b.closed = true; return nil }

func TestNonReplayableRequestAndResponseRemainLazy(t *testing.T) {
	originalRequest := &trackedBody{reader: strings.NewReader("secret-stream")}
	originalResponse := &trackedBody{reader: strings.NewReader(`{"name":"stream-visible"}`)}
	req := debugRequest(t, "POST", "https://example.test", "", "application/json")
	req.Body, req.GetBody = originalRequest, nil
	var trace bytes.Buffer
	response, err := httpdebug.New(roundTripFunc(func(got *http.Request) (*http.Response, error) {
		if got.Body != originalRequest || originalRequest.reads != 0 || originalRequest.closed {
			t.Fatal("tracing consumed or replaced non-replayable request")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: originalResponse}, nil
	}), &trace).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if originalResponse.reads != 0 || originalResponse.closed {
		t.Fatal("RoundTrip eagerly read response stream")
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if !originalResponse.closed || originalResponse.reads != 0 {
		t.Fatal("Close drained response or failed to close it")
	}
	assertHidden(t, trace.String(), "secret-stream", "stream-visible")
	if !strings.Contains(trace.String(), "non-replayable") {
		t.Fatal("non-replayable request omission not reported")
	}
}

type failingBody struct{ err error }

func (b failingBody) Read(p []byte) (int, error) { return copy(p, "secret-partial"), b.err }
func (b failingBody) Close() error               { return b.err }

func TestTransportAndBodyErrorsArePreservedButNotPrinted(t *testing.T) {
	want := errors.New("secret-error https://example.test?token=secret-token")
	req := debugRequest(t, "GET", "https://example.test", "", "")
	var trace bytes.Buffer
	response, err := httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, want
	}), &trace).RoundTrip(req)
	if response != nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("transport error unexpectedly returned a response")
	}
	if !errors.Is(err, want) {
		t.Fatal("changed transport error")
	}
	response, err = httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Header: http.Header{"Content-Type": {"application/json"}}, Body: failingBody{want}}, nil
	}), &trace).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	if string(data) != "secret-partial" || !errors.Is(err, want) {
		t.Fatal("changed partial body or read error")
	}
	if err := response.Body.Close(); !errors.Is(err, want) {
		t.Fatal("changed body close error")
	}
	assertHidden(t, trace.String(), "secret-")
}

func TestOAuthErrorCodesRemainVisibleWithoutErrorText(t *testing.T) {
	for _, kind := range []string{"application/json", "application/x-www-form-urlencoded"} {
		for _, code := range []string{
			"invalid_grant", "invalid_client", "invalid_scope", "invalid_request",
			"access_denied", "expired_token", "unauthorized_client", "unsupported_grant_type",
			"authorization_pending", "slow_down", "server_error", "temporarily_unavailable", "secret-arbitrary-code",
		} {
			t.Run(kind+"/"+code, func(t *testing.T) { checkOAuthErrorTrace(t, kind, code) })
		}
	}
}

func checkOAuthErrorTrace(t *testing.T, kind, code string) {
	t.Helper()
	body := fmt.Sprintf(`{"error":%q,"error_description":"secret-echoed-description","nested":{"error":"secret-nested-error"}}`, code)
	if kind == "application/x-www-form-urlencoded" {
		body = url.Values{"error": {code}, "error_description": {"secret-echoed-description"}}.Encode()
	}
	var trace bytes.Buffer
	response, err := httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(body, kind), nil
	}), &trace).RoundTrip(debugRequest(t, "POST", "https://example.test/token", body, kind))
	if err != nil {
		t.Fatal(err)
	}
	if got := consumeResponse(t, response); got != body {
		t.Fatal("OAuth error tracing changed the HTTP body")
	}
	assertHidden(t, trace.String(), "secret-")
	if code != "secret-arbitrary-code" && !strings.Contains(trace.String(), code) {
		t.Fatalf("recognized OAuth error code was hidden: %s", &trace)
	}
}

func TestConcurrentTracesUseSerializedWriter(t *testing.T) {
	var trace bytes.Buffer
	transport := httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(`{"ok":true}`, "application/json"), nil
	}), &trace)
	req := debugRequest(t, "GET", "https://example.test", "", "")
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			response, err := transport.RoundTrip(req.Clone(t.Context()))
			if err != nil {
				t.Error(err)
				return
			}
			consumeResponse(t, response)
		})
	}
	wg.Wait()
	if strings.Count(trace.String(), "response body:") != 32 || strings.Count(trace.String(), "request") != 32 {
		t.Fatal("concurrent traces missing or interleaved")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("output unavailable") }

func TestWriterFailureDoesNotChangeHTTPResponse(t *testing.T) {
	req := debugRequest(t, "GET", "https://example.test", "", "")
	response, err := httpdebug.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(`{"ok":true}`, "application/json"), nil
	}), failingWriter{}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := consumeResponse(t, response); got != `{"ok":true}` {
		t.Fatal("failed trace output affected HTTP result")
	}
}
