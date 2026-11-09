package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const (
	CompletionTimeout  = 2 * time.Second
	CompletionPageSize = int32(100)
	CompletionMaxPages = 5
)

// CompletionClient uses only an existing, unexpired stack token and never prompts.
func CompletionClient(cmd *cobra.Command) (*sdk.FormanceV1, *cobra.Command, context.CancelFunc, bool) {
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, CompletionTimeout)
	completionCommand := *cmd
	completionCommand.SetContext(ctx)
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(&completionCommand)
	if err != nil || profile == nil {
		cancel()
		return nil, nil, func() {}, false
	}
	organizationID, stackID, err := fctl.ResolveStackID(&completionCommand, *profile)
	if err != nil {
		cancel()
		return nil, nil, func() {}, false
	}
	token, err := fctl.ReadStackToken(&completionCommand, profileName, organizationID, stackID)
	if err != nil || token == nil || token.Expired() {
		cancel()
		return nil, nil, func() {}, false
	}
	client, err := fctl.NewStackClientFromFlags(&completionCommand, relyingParty, silentDialog{}, profileName, *profile)
	if err != nil || client == nil || client.Connectivity == nil || client.Connectivity.V1 == nil {
		cancel()
		return nil, nil, func() {}, false
	}
	return client.Connectivity.V1, &completionCommand, cancel, true
}

type silentDialog struct{}

func (silentDialog) Info(string, ...any) {}

// CompleteFilterExpressions completes `--filter` expressions from the
// server's query allowlist: keys and operators come from
// `GET /_query/capabilities`, values from the capability enum, from
// `GET /connectors/_facets` for connector tags, from the published connectors
// for the connector `catalog` key, and from the published connector names for
// the instance `connector` key. Connector tag and catalog counts honour the
// `--filter` expressions already on the command line.
func CompleteFilterExpressions(resource string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := CompletionClient(cmd)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()
		existing, _ := cmd.Flags().GetStringArray(FilterFlag)
		return completeFilterExpressionsWithin(completionCommand.Context(), client, resource, toComplete, existing)

	}
}

func completeFilterExpressions(ctx context.Context, client *sdk.FormanceV1, resource, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeFilterExpressionsWithin(ctx, client, resource, toComplete, nil)
}

// completeFilterExpressionsWithin narrows value counts to the connectors the
// existing filters already select; an invalid existing filter is ignored.
func completeFilterExpressionsWithin(ctx context.Context, client *sdk.FormanceV1, resource, toComplete string, existing []string) ([]string, cobra.ShellCompDirective) {

	capabilities, err := client.GetQueryCapabilities(ctx, nil)
	if err != nil || capabilities == nil || capabilities.QueryCapabilities == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	fields, ok := capabilities.QueryCapabilities.Resources[resource]
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
	scope, err := BuildListQuery("", existing)
	if err != nil {
		scope = ""
	}
	values := filterValueCandidates(ctx, client, resource, key, capability, scope)
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

// CompleteChannels serves the maturity channels from the server's capability
// enum for the instance `channel` key, so the CLI never hardcodes them.
func CompleteChannels() cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := CompletionClient(cmd)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		return completeChannels(completionCommand.Context(), client, toComplete)

	}
}

func completeChannels(ctx context.Context, client *sdk.FormanceV1, toComplete string) ([]string, cobra.ShellCompDirective) {
	capabilities, err := client.GetQueryCapabilities(ctx, nil)
	if err != nil || capabilities == nil || capabilities.QueryCapabilities == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	channels := capabilities.QueryCapabilities.Resources[ResourceConnections]["channel"].Enum
	candidates := make([]string, 0, len(channels))
	for _, channel := range channels {
		if strings.HasPrefix(channel, toComplete) {
			candidates = append(candidates, channel)
		}
	}
	sort.Strings(candidates)
	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func completeFilterKeys(fields map[string]connectivity.QueryFieldCapability, prefix string) []string {
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

func filterValueCandidates(ctx context.Context, client *sdk.FormanceV1, resource, key string, capability connectivity.QueryFieldCapability, scope string) []filterValue {
	if len(capability.Enum) > 0 {
		values := make([]filterValue, 0, len(capability.Enum))
		for _, value := range capability.Enum {
			values = append(values, filterValue{value: value})
		}
		return values
	}
	switch {
	case resource == ResourceConnectors && key == "tags":
		return facetFilterValues(ctx, client, scope)
	case resource == ResourceConnectors && key == "catalog":
		return catalogFilterValues(ctx, client, scope)
	case resource == ResourceConnections && key == "connector":
		return connectorFilterValues(ctx, client)
	}
	return nil
}

func facetFilterValues(ctx context.Context, client *sdk.FormanceV1, scope string) []filterValue {
	facets, err := client.GetConnectorFacets(ctx, operations.GetConnectorFacetsRequest{Query: OptionalString(scope)}, nil)
	if err != nil || facets == nil || facets.FacetDistribution == nil {
		return nil
	}
	values := make([]filterValue, 0)
	for facet, counts := range facets.FacetDistribution.Facets {
		for value, count := range counts {
			plural := "s"
			if count == 1 {
				plural = ""
			}
			if count == 0 {
				continue
			}
			values = append(values, filterValue{
				value:       facet + ":" + value,
				description: fmt.Sprintf("%d connector%s", count, plural),
			})
		}
	}
	return values
}

// catalogFilterValues counts the published connectors of each catalog.
func catalogFilterValues(ctx context.Context, client *sdk.FormanceV1, scope string) []filterValue {
	connectors, err := completionConnectors(ctx, client, scope)
	if err != nil {
		return nil
	}
	counts := map[string]int{}
	for _, connector := range connectors {
		if catalog := fctl.StringPointerToString(connector.Catalog); catalog != "" {
			counts[catalog]++
		}
	}
	values := make([]filterValue, 0, len(counts))
	for catalog, count := range counts {
		plural := "s"
		if count == 1 {
			plural = ""
		}
		values = append(values, filterValue{value: catalog, description: fmt.Sprintf("%d connector%s", count, plural)})
	}
	return values
}

func connectorFilterValues(ctx context.Context, client *sdk.FormanceV1) []filterValue {
	connectors, err := completionConnectors(ctx, client, "")
	if err != nil {
		return nil
	}
	values := make([]filterValue, 0, len(connectors))
	for _, connector := range connectors {
		if connector.Name == "" {
			continue
		}
		values = append(values, filterValue{value: connector.Name, description: ConnectorSummary(connector)})
	}
	return values
}

// ConnectorSummary is the one-line description shown next to a connector
// name in completions: the tagline, else the display name.
func ConnectorSummary(connector connectivity.Connector) string {
	if tagline := fctl.StringPointerToString(connector.Tagline); tagline != "" {
		return tagline
	}
	return fctl.StringPointerToString(connector.DisplayName)
}

func completionConnectors(ctx context.Context, client *sdk.FormanceV1, query string) ([]connectivity.Connector, error) {
	return CollectPagesBounded(CompletionPageSize, CompletionMaxPages,
		func(options ListOptions) ([]connectivity.Connector, bool, string, error) {
			size := int(options.PageSize)
			page, err := client.ListConnectors(ctx, operations.ListConnectorsRequest{PageSize: &size, Cursor: OptionalString(options.Cursor), Query: OptionalString(query)}, nil)
			if err != nil {
				return nil, false, "", err
			}
			if page == nil || page.ConnectorCursor == nil {
				return nil, false, "", fmt.Errorf("empty connector completion response")
			}
			cursor := page.ConnectorCursor.Cursor
			if cursor.Data == nil {
				return nil, false, "", fmt.Errorf("empty connector completion page")
			}
			next := ""
			if cursor.Next != nil {
				next = *cursor.Next
			}
			return cursor.Data, cursor.HasMore, next, nil
		})
}
