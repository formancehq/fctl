package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

const (
	CompletionTimeout  = 2 * time.Second
	CompletionPageSize = int32(100)
)

// CompletionClient builds a non-interactive, deadline-bounded client for
// shell completion callbacks, on a copy of the command so the original
// context is left untouched.
func CompletionClient(cmd *cobra.Command, factory ClientFactory) (connectivityclient.Client, *cobra.Command, context.CancelFunc, bool) {
	if factory == nil {
		return nil, nil, func() {}, false
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, CompletionTimeout)
	completionCommand := *cmd
	completionCommand.SetContext(WithNonInteractive(ctx))
	client, err := factory(&completionCommand)
	if err != nil || client == nil {
		cancel()
		return nil, nil, func() {}, false
	}
	return client, &completionCommand, cancel, true
}

// CompleteFilterExpressions completes `--filter` expressions from the
// server's query allowlist: keys and operators come from
// `GET /_query/capabilities`, values from the capability enum, from
// `GET /connectors/_facets` for connector tags, and from the published
// connector names for the instance `connector` key.
func CompleteFilterExpressions(factory ClientFactory, resource string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := CompletionClient(cmd, factory)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()
		ctx := completionCommand.Context()

		capabilities, err := client.GetQueryCapabilities(ctx)
		if err != nil || capabilities == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		fields, ok := capabilities.Resources[resource]
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		key, valuePrefix, operator := splitFilter(toComplete)
		if operator == "" {
			return completeFilterKeys(fields, toComplete), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
		}

		capability, ok := fields[key]
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		values := filterValueCandidates(ctx, client, resource, key, capability)
		candidates := make([]string, 0, len(values))
		for _, value := range values {
			if !strings.HasPrefix(value.value, valuePrefix) {
				continue
			}
			candidate := key + operator + value.value
			if value.description != "" {
				candidate += "\t" + value.description
			}
			candidates = append(candidates, candidate)
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteChannels serves the maturity channels from the server's capability
// enum for the instance `channel` key, so the CLI never hardcodes them.
func CompleteChannels(factory ClientFactory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := CompletionClient(cmd, factory)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		capabilities, err := client.GetQueryCapabilities(completionCommand.Context())
		if err != nil || capabilities == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		channels := capabilities.Resources[connectivityclient.ResourceConnectorInstances]["channel"].Enum
		candidates := make([]string, 0, len(channels))
		for _, channel := range channels {
			if strings.HasPrefix(channel, toComplete) {
				candidates = append(candidates, channel)
			}
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeFilterKeys(fields map[string]connectivityclient.QueryFieldCapability, prefix string) []string {
	candidates := make([]string, 0, len(fields))
	for key, capability := range fields {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		candidate := key + "="
		if len(capability.Operators) > 0 {
			candidate += "\toperators: " + strings.Join(capability.Operators, ", ")
		}
		candidates = append(candidates, candidate)
	}
	sort.Strings(candidates)
	return candidates
}

type filterValue struct {
	value       string
	description string
}

func filterValueCandidates(ctx context.Context, client connectivityclient.Client, resource, key string, capability connectivityclient.QueryFieldCapability) []filterValue {
	if len(capability.Enum) > 0 {
		values := make([]filterValue, 0, len(capability.Enum))
		for _, value := range capability.Enum {
			values = append(values, filterValue{value: value})
		}
		return values
	}
	switch {
	case resource == connectivityclient.ResourceConnectors && key == "tags":
		return facetFilterValues(ctx, client)
	case resource == connectivityclient.ResourceConnectorInstances && key == "connector":
		return connectorFilterValues(ctx, client)
	}
	return nil
}

func facetFilterValues(ctx context.Context, client connectivityclient.Client) []filterValue {
	facets, err := client.GetConnectorFacets(ctx, "")
	if err != nil || facets == nil {
		return nil
	}
	values := make([]filterValue, 0)
	for facet, counts := range facets.Facets {
		for value, count := range counts {
			plural := "s"
			if count == 1 {
				plural = ""
			}
			values = append(values, filterValue{
				value:       facet + ":" + value,
				description: fmt.Sprintf("%d connector%s", count, plural),
			})
		}
	}
	return values
}

func connectorFilterValues(ctx context.Context, client connectivityclient.Client) []filterValue {
	connectors, err := client.ListConnectors(ctx, connectivityclient.ListOptions{PageSize: CompletionPageSize})
	if err != nil || connectors == nil {
		return nil
	}
	values := make([]filterValue, 0, len(connectors.Items))
	for _, connector := range connectors.Items {
		if connector.Metadata.Name == nil || *connector.Metadata.Name == "" {
			continue
		}
		value := filterValue{value: *connector.Metadata.Name}
		if connector.Spec.DisplayName != nil {
			value.description = *connector.Spec.DisplayName
		}
		values = append(values, value)
	}
	return values
}
