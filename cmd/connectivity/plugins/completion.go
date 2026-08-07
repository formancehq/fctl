package plugins

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

const (
	pluginCompletionLimit   = int32(500)
	pluginCompletionTimeout = 2 * time.Second
)

func CompletePluginNames(factory connectivityinternal.ClientFactory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if factory == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		parent := cmd.Context()
		if parent == nil {
			parent = context.Background()
		}
		ctx, cancel := context.WithTimeout(parent, pluginCompletionTimeout)
		defer cancel()

		completionCommand := *cmd
		completionCommand.SetContext(ctx)
		client, err := factory(&completionCommand)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		response, err := client.ListPlugins(ctx, connectivityclient.ListOptions{Limit: pluginCompletionLimit})
		if err != nil || response == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		candidates := make([]string, 0, len(response.Items))
		for _, plugin := range response.Items {
			name := stringValue(plugin.Metadata.Name)
			if name == "" || !strings.HasPrefix(name, toComplete) {
				continue
			}
			if description := stringValue(plugin.Spec.Description); description != "" {
				name += "\t" + description
			}
			candidates = append(candidates, name)
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
}
