package stack

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDemoListThroughStackCommand(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		t.Run(format, func(t *testing.T) {
			cmd := NewCommand()
			cmd.PersistentFlags().String("output", format, "")
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"demo", "--list"})
			require.NoError(t, cmd.ExecuteContext(t.Context()))
			require.Empty(t, stderr.String())
			if format == "plain" {
				require.Contains(t, stdout.String(), "omnibus")
			} else {
				var result struct {
					Data struct {
						Recipes []struct {
							Name        string `json:"name"`
							Description string `json:"description"`
						} `json:"recipes"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
				require.Len(t, result.Data.Recipes, 10)
				names := make([]string, 0, len(result.Data.Recipes))
				for _, recipe := range result.Data.Recipes {
					names = append(names, recipe.Name)
					require.NotEmpty(t, recipe.Description)
				}
				require.Contains(t, names, "omnibus")
				require.Contains(t, names, "rideshare-tutorial")
				var raw struct {
					Data map[string]any `json:"data"`
				}
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &raw))
				require.Len(t, raw.Data, 1, "list JSON must not contain run counters")
			}
		})
	}
}
