package connections

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const instanceCompletionMaxPages = 5

type PathCompleter func(prefix string) ([]string, error)

// ConnectorResolver names the Connector a command is operating on.
type ConnectorResolver func(context.Context, *sdk.FormanceV1, *cobra.Command, []string) (string, error)

// ConnectorVersionResolver resolves the ConnectorVersion whose configSchema
// drives value completion for a command.
type ConnectorVersionResolver func(context.Context, *sdk.FormanceV1, *cobra.Command, []string) (*connectivity.ConnectorVersion, error)

func CompleteConnectionNames() cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := connectivityinternal.CompletionClient(cmd)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		return completeConnectionNamesWithClient(client, completionCommand, toComplete)
	}
}

func completeConnectionNamesWithClient(client *sdk.FormanceV1, completionCommand *cobra.Command, toComplete string) ([]string, cobra.ShellCompDirective) {
	instances, err := connectivityinternal.CollectPagesBounded(connectivityinternal.CompletionPageSize, instanceCompletionMaxPages,
		func(options connectivityinternal.ListOptions) ([]connectivity.Connection, bool, string, error) {
			response, err := client.ListConnections(completionCommand.Context(), operations.ListConnectionsRequest{Cursor: connectivityinternal.OptionalString(options.Cursor), PageSize: fctl.Ptr(int(options.PageSize)), Query: connectivityinternal.OptionalString(options.Query)}, nil)
			var page *connectivity.ConnectionCursor
			if response != nil {
				page = response.ConnectionCursor
			}
			if err != nil || page == nil {
				return nil, false, "", err
			}
			return page.Cursor.Data, page.Cursor.HasMore, fctl.StringPointerToString(page.Cursor.Next), nil
		})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	candidates := make([]string, 0, len(instances))
	for _, instance := range instances {
		name := instance.Name
		if name == "" || !strings.HasPrefix(name, toComplete) {
			continue
		}
		description := strings.Join(nonEmptyStrings(fctl.StringPointerToString(instance.ConnectionConfiguration.Connector), fctl.StringPointerToString(instance.ConnectionConfiguration.Ledger)), " · ")
		if description != "" {
			name += "\t" + description
		}
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)
	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func CompleteVersions(connectorArg func(*cobra.Command, []string) string) cobra.CompletionFunc {
	return completeVersions(func(_ context.Context, _ *sdk.FormanceV1, cmd *cobra.Command, args []string) (string, error) {
		if connectorArg == nil {
			return "", nil
		}
		return connectorArg(cmd, args), nil
	})
}

func completeVersions(resolveConnector ConnectorResolver) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if resolveConnector == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		client, completionCommand, cancel, ok := connectivityinternal.CompletionClient(cmd)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		return completeVersionsWithClient(client, completionCommand, args, toComplete, resolveConnector)
	}
}

func completeVersionsWithClient(client *sdk.FormanceV1, completionCommand *cobra.Command, args []string, toComplete string, resolveConnector ConnectorResolver) ([]string, cobra.ShellCompDirective) {
	if resolveConnector == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	connector, err := resolveConnector(completionCommand.Context(), client, completionCommand, args)
	if err != nil || connector == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	versions, err := connectivityinternal.CollectPagesBounded(connectivityinternal.CompletionPageSize, instanceCompletionMaxPages,
		func(options connectivityinternal.ListOptions) ([]connectivity.ConnectorVersionSummary, bool, string, error) {
			response, err := client.ListConnectorVersions(completionCommand.Context(), operations.ListConnectorVersionsRequest{Name: url.PathEscape(connector), Cursor: connectivityinternal.OptionalString(options.Cursor), PageSize: fctl.Ptr(int(options.PageSize)), Query: connectivityinternal.OptionalString(options.Query)}, nil)
			var page *connectivity.ConnectorVersionCursor
			if response != nil {
				page = response.ConnectorVersionCursor
			}
			if err != nil || page == nil {
				return nil, false, "", err
			}
			return page.Cursor.Data, page.Cursor.HasMore, fctl.StringPointerToString(page.Cursor.Next), nil
		})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	candidates := make([]string, 0, len(versions))
	for _, version := range versions {
		if version.Version == "" || !strings.HasPrefix(version.Version, toComplete) {
			continue
		}
		candidate := version.Version
		description := fctl.StringPointerToString(version.Digest)
		if description != "" {
			candidate += "\t" + description
		}
		candidates = append(candidates, candidate)
	}
	sort.Strings(candidates)
	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func CompleteSetValues(
	resolveVersion ConnectorVersionResolver,
	paths PathCompleter,
) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if strings.Contains(toComplete, "=@") || resolveVersion == nil {
			return completeSetValuesWithClient(nil, cmd, args, toComplete, resolveVersion, paths)
		}

		client, completionCommand, cancel, ok := connectivityinternal.CompletionClient(cmd)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		return completeSetValuesWithClient(client, completionCommand, args, toComplete, resolveVersion, paths)
	}
}

func completeSetValuesWithClient(client *sdk.FormanceV1, completionCommand *cobra.Command, args []string, toComplete string, resolveVersion ConnectorVersionResolver, paths PathCompleter) ([]string, cobra.ShellCompDirective) {
	if marker := strings.Index(toComplete, "=@"); marker >= 0 {
		if paths == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		candidatePrefix := toComplete[:marker+2]
		pathPrefix := toComplete[marker+2:]
		pathCandidates, err := paths(pathPrefix)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		candidates := make([]string, 0, len(pathCandidates))
		for _, path := range pathCandidates {
			if strings.HasPrefix(path, pathPrefix) {
				candidates = append(candidates, candidatePrefix+path)
			}
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}

	if resolveVersion == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	version, err := resolveVersion(completionCommand.Context(), client, completionCommand, args)
	if err != nil || version == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	fields, err := configuration.SchemaFields(version)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	supplied := suppliedSetKeys(completionCommand, args)
	ordered := make([]configuration.SchemaField, 0, len(fields))
	for key, field := range fields {
		if supplied[key] || !strings.HasPrefix(key, strings.TrimSuffix(toComplete, "=")) {
			continue
		}
		ordered = append(ordered, field)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Required != ordered[j].Required {
			return ordered[i].Required
		}
		return ordered[i].Key < ordered[j].Key
	})
	candidates := make([]string, 0, len(ordered))
	for _, field := range ordered {
		candidates = append(candidates, field.Key+"=\t"+schemaFieldDescription(field))
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func schemaFieldDescription(field configuration.SchemaField) string {
	if field.Description != "" {
		return field.Description
	}
	if field.Kind == configuration.ConfigFile {
		return "file configuration"
	}
	return "environment configuration"
}

func OSPathCompleter(prefix string) ([]string, error) {
	directory, base := filepath.Split(prefix)
	readDirectory := directory
	if readDirectory == "" {
		readDirectory = "."
	}
	entries, err := os.ReadDir(readDirectory)
	if err != nil {
		return nil, err
	}
	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), base) {
			continue
		}
		candidate := directory + entry.Name()
		if entry.IsDir() {
			candidate += string(filepath.Separator)
		}
		candidates = append(candidates, candidate)
	}
	sort.Strings(candidates)
	return candidates, nil
}

func suppliedSetKeys(cmd *cobra.Command, args []string) map[string]bool {
	values := append([]string(nil), args...)
	if cmd != nil && cmd.Flags().Lookup("set") != nil {
		if setValues, err := cmd.Flags().GetStringArray("set"); err == nil {
			values = append(values, setValues...)
		}
	}
	supplied := make(map[string]bool)
	for _, value := range values {
		if separator := strings.IndexByte(value, '='); separator > 0 {
			supplied[value[:separator]] = true
		}
	}
	return supplied
}

func nonEmptyStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
