package connectivityclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type sdkContextKey struct{}

type observedBody struct {
	reader  io.Reader
	readErr error
	reads   int
	closes  int
}

func (b *observedBody) Read(p []byte) (int, error) {
	b.reads++
	n, err := b.reader.Read(p)
	if b.readErr != nil {
		err, b.readErr = b.readErr, nil
	}
	return n, err
}
func (b *observedBody) Close() error { b.closes++; return nil }

func TestSDKAdapterPreservesProvidedClientAndCallerContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.WithValue(t.Context(), sdkContextKey{}, "caller"), time.Minute)
	defer cancel()
	calls := 0
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Same(t, ctx, req.Context())
		require.Equal(t, "caller", req.Context().Value(sdkContextKey{}))
		require.Contains(t, req.Header.Get("User-Agent"), "speakeasy-sdk/go")
		require.Equal(t, "/tenant%2Fstack/api/connectivity/connections/worker%2Fone", req.URL.EscapedPath())
		return jsonResponse(http.StatusOK, instanceResponse), nil
	})
	_, err := New("https://stack.example/tenant%2Fstack/?discard=yes#fragment", &http.Client{Transport: transport}).GetConnection(ctx, "worker/one")
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

func TestSDKAdapterClosesOriginalBodyAndRetainsReadFailures(t *testing.T) {
	readErr := errors.New("response stream failed")
	for _, tt := range []struct {
		name, body string
		status     int
		readErr    error
		decodeErr  bool
		noContent  bool
	}{
		{name: "success", body: instanceResponse, status: 200},
		{name: "malformed success", body: `{`, status: 200, decodeErr: true},
		{name: "valid JSON with read failure", body: instanceResponse, status: 200, readErr: readErr},
		{name: "empty read failure", status: 200, readErr: readErr},
		{name: "structured error with read failure", body: `{"code":"forbidden","message":"denied"}`, status: 403, readErr: readErr},
		{name: "structured error", body: `{"code":"forbidden","message":"denied","details":{"field":"name"}}`, status: 403},
		{name: "malformed declared error", body: `{"code":`, status: 403},
		{name: "empty unauthorized", status: 401},
		{name: "no content", status: 204, noContent: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &observedBody{reader: strings.NewReader(tt.body), readErr: tt.readErr}
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				r := jsonResponse(tt.status, "")
				r.Body = body
				return r, nil
			})})
			var err error
			if tt.noContent {
				err = c.DeleteConnection(t.Context(), "worker")
			} else {
				_, err = c.GetConnection(t.Context(), "worker")
			}
			require.Equal(t, 1, body.closes)
			switch {
			case tt.noContent:
				require.NoError(t, err)
				require.Zero(t, body.reads)
			case tt.readErr != nil:
				require.ErrorIs(t, err, tt.readErr)
				require.ErrorContains(t, err, "read connectivity response")
			case tt.decodeErr:
				require.ErrorContains(t, err, "decode connectivity response")
			case tt.status != 200:
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
				require.Equal(t, tt.status, apiErr.StatusCode)
				if tt.name == "structured error" {
					require.Equal(t, "forbidden", apiErr.Code)
					require.Equal(t, map[string]any{"field": "name"}, apiErr.Details)
				} else {
					require.Empty(t, apiErr.Code)
					require.Equal(t, http.StatusText(tt.status), apiErr.Message)
				}
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestSDKAdapterValidatesOriginalResponseBytes(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		valid      bool
	}{
		{"missing progress", `{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active","suspendedBy":[],"ingestion":{"progress":{}}}}`, false},
		{"null progress value", `{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active","suspendedBy":[],"ingestion":{"progress":{"current":null}}}}`, false},
		{"recorded zero", `{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active","suspendedBy":[],"ingestion":{"progress":{"current":0}}}}`, true},
		{"missing suspension name", `{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"suspended","suspendedBy":[{"source":"request"}]}}`, false},
		{"missing diagnostic message", `{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"failed","suspendedBy":[],"error":{"code":"service_error"}}}`, false},
		{"invalid cursor", `{"name":"worker","configuration":{"ledger":"main","initialCursor":"***"}}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				r := jsonResponse(200, tt.body)
				r.Header.Set("Content-Type", "text/plain")
				return r, nil
			})})
			got, err := c.GetConnection(t.Context(), "worker")
			if tt.valid {
				require.NoError(t, err)
				require.Zero(t, got.Status.Ingestion.Progress.Current)
			} else {
				require.Nil(t, got)
				require.ErrorContains(t, err, "decode connectivity response")
			}
		})
	}
}

func TestSDKAdapterOmitsUnspecifiedSizeOnCursorRequests(t *testing.T) {
	options := ListOptions{Cursor: "opaque/+?=", Query: ` {"$gte":{"custom":12}} `}
	for _, tt := range []struct {
		name string
		call func(Client) error
	}{
		{"connectors", func(c Client) error { _, e := c.ListConnectors(t.Context(), options); return e }},
		{"versions", func(c Client) error { _, e := c.ListConnectorVersions(t.Context(), "stripe", options); return e }},
		{"connections", func(c Client) error { _, e := c.ListConnections(t.Context(), options); return e }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				require.NotContains(t, r.URL.Query(), "pageSize")
				require.Equal(t, options.Cursor, r.URL.Query().Get("cursor"))
				require.Equal(t, options.Query, r.URL.Query().Get("query"))
				return jsonResponse(200, emptyPage), nil
			})})
			require.NoError(t, tt.call(c))
		})
	}
}

func TestSDKAdapterConcurrentCallsKeepResponseOwnership(t *testing.T) {
	const count = 16
	arrived := make(chan struct{}, count)
	release := make(chan struct{})
	results := make(chan error, count)
	bodies := make([]*observedBody, count)
	c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		name := strings.TrimPrefix(r.URL.Path, "/api/connectivity/connections/")
		index, err := strconv.Atoi(strings.TrimPrefix(name, "worker-"))
		if err != nil {
			return nil, err
		}
		if r.Context().Value(sdkContextKey{}) != name {
			return nil, fmt.Errorf("caller context lost for %s", name)
		}
		arrived <- struct{}{}
		<-release
		status := 200
		payload := fmt.Sprintf(`{"name":%q,"configuration":{"ledger":"main"}}`, name)
		if index%2 == 0 {
			status = 403
			payload = fmt.Sprintf(`{"code":"forbidden","message":"denied","details":{"name":%q}}`, name)
		}
		body := &observedBody{reader: strings.NewReader(payload)}
		bodies[index] = body
		response := jsonResponse(status, "")
		response.Body = body
		return response, nil
	})})
	for i := range count {
		go func() {
			name := fmt.Sprintf("worker-%d", i)
			ctx := context.WithValue(t.Context(), sdkContextKey{}, name)
			got, err := c.GetConnection(ctx, name)
			if i%2 == 0 {
				var apiErr *APIError
				if got != nil || !errors.As(err, &apiErr) || apiErr.StatusCode != 403 || apiErr.Details["name"] != name {
					results <- fmt.Errorf("mixed error for %s: %v", name, err)
					return
				}
			} else if err != nil || got == nil || got.Name != name {
				results <- fmt.Errorf("mixed response for %s: %+v %v", name, got, err)
				return
			}
			results <- nil
		}()
	}
	for range count {
		<-arrived
	}
	close(release)
	for range count {
		require.NoError(t, <-results)
	}
	for _, body := range bodies {
		require.Equal(t, 1, body.closes)
	}
}

func TestSDKAdapterEscapesReservedCharactersInPathArguments(t *testing.T) {
	for _, name := range []string{"worker/one", "literal%2Fname", "name?query=value#fragment", "name with spaces+plus"} {
		t.Run(name, func(t *testing.T) {
			transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "/api/connectivity/connectors/"+url.PathEscape(name)+"/versions/"+url.PathEscape(name), r.URL.EscapedPath())
				require.Empty(t, r.URL.RawQuery)
				require.Empty(t, r.URL.Fragment)
				return jsonResponse(200, `{"version":"v1"}`), nil
			})
			_, err := New("https://stack.example", &http.Client{Transport: transport}).GetConnectorVersion(t.Context(), name, name)
			require.NoError(t, err)
		})
	}
}
