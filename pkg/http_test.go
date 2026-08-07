package fctl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
