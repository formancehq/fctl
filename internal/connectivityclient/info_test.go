package connectivityclient

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const serviceInfoResponse = `{"version":"v1.2.3","commit":"abc123","buildDate":"2026-10-28T22:29:14Z"}`

func TestGetInfoUsesSDKAndCallerContextWithOrWithoutDebug(t *testing.T) {
	for _, body := range []string{serviceInfoResponse, strings.TrimSuffix(serviceInfoResponse, "}") + `,"debug":true}`} {
		t.Run(body, func(t *testing.T) {
			ctx := context.WithValue(t.Context(), sdkContextKey{}, "info caller")
			calls := 0
			original := &observedBody{reader: strings.NewReader(body)}
			c := New("https://stack.example/base/?discard=true#fragment", &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Same(t, ctx, req.Context())
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "/base/api/connectivity/_info", req.URL.EscapedPath())
				require.Empty(t, req.URL.RawQuery)
				require.Empty(t, req.URL.Fragment)
				require.Contains(t, req.Header.Get("User-Agent"), "speakeasy-sdk/go")
				require.Nil(t, req.Body)
				r := jsonResponse(200, "")
				r.Body = original
				return r, nil
			})})
			info, err := c.GetInfo(ctx)
			require.NoError(t, err)
			require.Equal(t, &ServiceInfo{Version: "v1.2.3", Commit: "abc123", BuildDate: "2026-10-28T22:29:14Z"}, info)
			data, err := json.Marshal(info)
			require.NoError(t, err)
			require.JSONEq(t, serviceInfoResponse, string(data))
			require.Equal(t, 1, calls)
			require.Equal(t, 1, original.closes)
		})
	}
}

func TestGetInfoRejectsMalformedOrMissingBuildMetadata(t *testing.T) {
	for _, body := range []string{
		"", `{`, `null`, `[]`, `{}`,
		`{"commit":"abc","buildDate":"today"}`, `{"version":"v1","buildDate":"today"}`, `{"version":"v1","commit":"abc"}`,
		`{"version":null,"commit":"abc","buildDate":"today"}`, `{"version":"v1","commit":null,"buildDate":"today"}`, `{"version":"v1","commit":"abc","buildDate":null}`,
		`{"version":1,"commit":"abc","buildDate":"today"}`, `{"version":"v1","commit":true,"buildDate":"today"}`, `{"version":"v1","commit":"abc","buildDate":{}}`,
	} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "/api/connectivity/_info", req.URL.Path)
				return jsonResponse(200, body), nil
			})})
			info, err := c.GetInfo(t.Context())
			require.Nil(t, info)
			require.ErrorContains(t, err, "decode connectivity response")
			require.Equal(t, 1, calls)
		})
	}
	// Presence is distinct from content: development builds may report empty strings.
	info, err := New("https://stack.example", responseClient(200, `{"version":"","commit":"","buildDate":""}`)).GetInfo(t.Context())
	require.NoError(t, err)
	require.Equal(t, &ServiceInfo{}, info)
}

func TestGetInfoNormalizesHTTPErrorsWithoutHealthFallback(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
	}{
		{"empty unauthorized", "", 401}, {"non JSON unavailable", "service unavailable", 503},
		{"structured", `{"code":"forbidden","message":"not allowed","details":{"resource":"info"}}`, 403},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "/api/connectivity/_info", req.URL.Path)
				return jsonResponse(tt.status, tt.body), nil
			})})
			info, err := c.GetInfo(t.Context())
			require.Nil(t, info)
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, tt.status, apiErr.StatusCode)
			if tt.name == "structured" {
				require.Equal(t, "forbidden", apiErr.Code)
				require.Equal(t, map[string]any{"resource": "info"}, apiErr.Details)
			} else {
				require.Equal(t, http.StatusText(tt.status), apiErr.Message)
			}
			require.Equal(t, 1, calls)
		})
	}
}

func TestGetInfoPropagatesCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c := New("https://stack.example", &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) { return nil, req.Context().Err() })})
	info, err := c.GetInfo(ctx)
	require.Nil(t, info)
	require.ErrorIs(t, err, context.Canceled)
}
