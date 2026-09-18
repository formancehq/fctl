package connectors

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

const connectorCompletionMaxPages = 5

func CompleteConnectorNames(factory connectivityinternal.ClientFactory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := connectivityinternal.CompletionClient(cmd, factory)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		connectors, err := connectivityinternal.CollectPagesBounded(connectivityinternal.CompletionPageSize, connectorCompletionMaxPages,
			func(options connectivityclient.ListOptions) ([]connectivityclient.Connector, bool, string, error) {
				page, err := client.ListConnectors(completionCommand.Context(), options)
				if err != nil || page == nil {
					return nil, false, "", err
				}
				return page.Items, page.HasMore, page.Next, nil
			})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		candidates := make([]string, 0, len(connectors))
		for _, connector := range connectors {
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
