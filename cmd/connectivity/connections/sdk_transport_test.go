package connections

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
)

func TestConfigureUsesSDKRouteAndPreservesMergePatchIntent(t *testing.T) {
	for _, tt := range []struct {
		name         string
		flags        []string
		config, want string
	}{
		{name: "pin replaces channel", flags: []string{"--version=v3.0.0"}, want: `{"configuration":{"version":"v3.0.0","channel":null}}`},
		{name: "channel replaces pin", flags: []string{"--channel=stable"}, want: `{"configuration":{"channel":"stable","version":null}}`},
		{name: "delete one entry", flags: []string{"--config=config.yaml"}, config: "env:\n  TOKEN: null\n  API_URL: replacement\n", want: `{"configuration":{"env":{"TOKEN":null,"API_URL":"replacement"}}}`},
		{name: "delete entire map", flags: []string{"--config=config.yaml"}, config: "env: null\nfiles: null\n", want: `{"configuration":{"env":null,"files":null}}`},
		{name: "empty map preserves values", flags: []string{"--config=config.yaml", "--poll-interval=1m"}, config: "env: {}\n", want: `{"configuration":{"pollInterval":"1m"}}`},
		{name: "opaque cursor", flags: []string{"--initial-cursor=AP9h"}, want: `{"configuration":{"initialCursor":"AP9h"}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			instance := instanceFixture("source/name?x")
			instanceJSON, err := json.Marshal(instance)
			require.NoError(t, err)
			versionJSON, err := json.Marshal(versionWithOptionalSchema())
			require.NoError(t, err)
			var methods []string
			api := connectivityinternal.NewAPI("https://stack.example.test/prefix", &http.Client{Transport: connectionRoundTripper(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "stack.example.test", r.URL.Host)
				if strings.Contains(r.URL.Path, "/connectors/") {
					require.Equal(t, http.MethodGet, r.Method)
					return jsonHTTPResponse(r, http.StatusOK, string(versionJSON)), nil
				}
				require.Equal(t, "/prefix/api/connectivity/connections/source%2Fname%3Fx", r.URL.EscapedPath())
				require.Empty(t, r.URL.RawQuery)
				methods = append(methods, r.Method)
				if r.Method == http.MethodPatch {
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.JSONEq(t, tt.want, string(body))
					require.Equal(t, int64(len(body)), r.ContentLength)
				}
				return jsonHTTPResponse(r, http.StatusOK, string(instanceJSON)), nil
			})})
			args := append([]string{"source/name?x", "--confirm"}, tt.flags...)
			_, err = executeCommand(NewConfigureCommand(factoryReturning(api), mockReadFile(map[string]string{"config.yaml": tt.config}), nil), args...)
			require.NoError(t, err)
			require.Equal(t, []string{http.MethodGet, http.MethodPatch}, methods)
		})
	}
}

func TestConnectionCommandsHandleSDKErrorsAndEmptySuccess(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     int
		body, want string
	}{
		{name: "typed error", status: 404, body: `{"code":"not_found","message":"fixture missing"}`, want: "fixture missing"},
		{name: "untyped error", status: 502, body: `{"diagnostic":"fixture detail"}`, want: "fixture detail"},
		{name: "empty success", status: 200, body: `null`, want: "empty response"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			api := connectivityinternal.NewAPI("https://stack.example.test", &http.Client{Transport: connectionRoundTripper(func(r *http.Request) (*http.Response, error) {
				response := jsonHTTPResponse(r, tt.status, tt.body)
				if tt.name == "untyped error" {
					response.Header.Set("Content-Type", "text/plain")
				}
				return response, nil
			})})
			_, err := executeCommand(NewShowCommand(factoryReturning(api)), "example")
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func jsonHTTPResponse(request *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}
}

func TestListConnectionsRejectsMissingSDKCursorData(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"cursor":{"data":null}}`} {
		api := connectivityinternal.NewAPI("https://stack.example.test", &http.Client{Transport: connectionRoundTripper(func(r *http.Request) (*http.Response, error) { return jsonHTTPResponse(r, http.StatusOK, body), nil })})
		_, err := executeCommand(NewListCommand(factoryReturning(api)))
		require.ErrorContains(t, err, "empty response")
	}
	api := connectivityinternal.NewAPI("https://stack.example.test", &http.Client{Transport: connectionRoundTripper(func(r *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(r, http.StatusOK, `{"cursor":{"data":[],"pageSize":15,"hasMore":false}}`), nil
	})})
	_, err := executeCommand(NewListCommand(factoryReturning(api)))
	require.NoError(t, err)
}

func TestSuspensionCommandsRejectEmptySDKSuccess(t *testing.T) {
	for _, suspend := range []bool{true, false} {
		action := "unsuspend"
		if suspend {
			action = "suspend"
		}
		for _, body := range []string{`null`, `{}`, `{"configuration":{}}`} {
			t.Run(action+"/"+body, func(t *testing.T) {
				api := connectivityinternal.NewAPI("https://stack.example.test", &http.Client{Transport: connectionRoundTripper(func(r *http.Request) (*http.Response, error) {
					require.Equal(t, http.MethodPatch, r.Method)
					require.Equal(t, "/api/connectivity/connections/example", r.URL.Path)
					return jsonHTTPResponse(r, http.StatusOK, body), nil
				})})
				command := NewUnsuspendCommand(factoryReturning(api))
				if suspend {
					command = NewSuspendCommand(factoryReturning(api))
				}
				output, err := executeCommand(command, "example", "--confirm")
				require.ErrorContains(t, err, "empty response")
				require.NotContains(t, output, "requested.")
			})
		}
	}
}
