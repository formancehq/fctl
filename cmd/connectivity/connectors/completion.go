package connectors

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const connectorCompletionMaxPages = 5

func CompleteConnectorNames() cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := connectivityinternal.CompletionClient(cmd)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		return completeConnectorNames(completionCommand, client, toComplete)
	}
}

func completeConnectorNames(completionCommand *cobra.Command, client *formance.FormanceV1, toComplete string) ([]string, cobra.ShellCompDirective) {

	connectors, err := connectivityinternal.CollectPagesBounded(connectivityinternal.CompletionPageSize, connectorCompletionMaxPages,
		func(options connectivityinternal.ListOptions) ([]connectivity.Connector, bool, string, error) {
			response, err := client.ListConnectors(completionCommand.Context(), operations.ListConnectorsRequest{PageSize: fctl.Ptr(int(options.PageSize)), Cursor: connectivityinternal.OptionalString(options.Cursor)}, nil)
			if err != nil {
				return nil, false, "", err
			}
			if response == nil {
				return nil, false, "", fmt.Errorf("empty connector list response")
			}
			if response.ConnectorCursor == nil {
				return nil, false, "", fmt.Errorf("empty connector list response")
			}
			page := response.ConnectorCursor.Cursor
			if page.Data == nil {
				return nil, false, "", fmt.Errorf("connector cursor.data must be an array")
			}
			return page.Data, page.HasMore, fctl.StringPointerToString(page.Next), nil
		})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	candidates := make([]string, 0, len(connectors))
	for _, connector := range connectors {
		name := connector.Name
		if name == "" || !strings.HasPrefix(name, toComplete) {
			continue
		}
		if description := fctl.StringPointerToString(connector.Description); description != "" {
			name += "\t" + description
		}
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)
	return candidates, cobra.ShellCompDirectiveNoFileComp
}
