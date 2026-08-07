package fctl

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
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

func TestRedactDebugBodyRemovesInlineConfigurationValues(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		want  string
		leaks []string
	}{
		{
			name: "API-shaped request config",
			body: `{
				"name":"worker",
				"spec":{
					"value":"public-spec-value",
					"config":{
						"env":{"PASSWORD":{"value":"request-env-secret"}},
						"files":[{"path":"/secret","value":"request-file-secret"}]
					}
				}
			}`,
			want: `{
				"name":"worker",
				"spec":{
					"value":"public-spec-value",
					"config":{
						"env":{"PASSWORD":{"value":"[REDACTED]"}},
						"files":[{"path":"/secret","value":"[REDACTED]"}]
					}
				}
			}`,
			leaks: []string{"request-env-secret", "request-file-secret"},
		},
		{
			name: "pinned CRD-shaped patch",
			body: `{
				"spec":{
					"env":{"PASSWORD":{"value":"patch-env-secret"}},
					"files":[{"path":"/secret","value":"patch-file-secret"}],
					"other":{"value":"public-other-value"}
				}
			}`,
			want: `{
				"spec":{
					"env":{"PASSWORD":{"value":"[REDACTED]"}},
					"files":[{"path":"/secret","value":"[REDACTED]"}],
					"other":{"value":"public-other-value"}
				}
			}`,
			leaks: []string{"patch-env-secret", "patch-file-secret"},
		},
		{
			name: "response config and plugin defaults",
			body: `{
				"items":[
					{"spec":{"config":{"env":{"TOKEN":{"value":"response-secret"}}}}},
					{"spec":{"defaults":{"files":[{"path":"/key","value":"default-secret"}]}}}
				]
			}`,
			want: `{
				"items":[
					{"spec":{"config":{"env":{"TOKEN":{"value":"[REDACTED]"}}}}},
					{"spec":{"defaults":{"files":[{"path":"/key","value":"[REDACTED]"}]}}}
				]
			}`,
			leaks: []string{"response-secret", "default-secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted := redactDebugBody([]byte(tt.body))

			require.JSONEq(t, tt.want, string(redacted))
			for _, leak := range tt.leaks {
				require.NotContains(t, string(redacted), leak)
			}
		})
	}
}

func TestRedactDebugBodyLeavesNonJSONUnchanged(t *testing.T) {
	body := []byte("not-json")

	require.Equal(t, body, redactDebugBody(body))
}
