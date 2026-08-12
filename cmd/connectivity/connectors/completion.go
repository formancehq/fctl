package connectors

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
	connectorCompletionLimit   = int32(500)
	connectorCompletionTimeout = 2 * time.Second
)

func CompleteConnectorNames(factory connectivityinternal.ClientFactory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if factory == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		parent := cmd.Context()
		if parent == nil {
			parent = context.Background()
		}
		ctx, cancel := context.WithTimeout(parent, connectorCompletionTimeout)
		defer cancel()

		completionCommand := *cmd
		completionCommand.SetContext(connectivityinternal.WithNonInteractive(ctx))
		client, err := factory(&completionCommand)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		response, err := client.ListConnectors(ctx, connectivityclient.ListOptions{Limit: connectorCompletionLimit})
		if err != nil || response == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		candidates := make([]string, 0, len(response.Items))
		for _, connector := range response.Items {
			name := stringValue(connector.Metadata.Name)
			if name == "" || !strings.HasPrefix(name, toComplete) {
				continue
			}
			if description := stringValue(connector.Spec.Description); description != "" {
				name += "\t" + description
			}
			candidates = append(candidates, name)
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
}
