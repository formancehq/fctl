package connections

import (
	"context"
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
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return connectivityinternal.CompleteConnectorVersions(completionCommand.Context(), client, connector, toComplete)
}

// isFileAssignment reports KEY=@path: only an @ right after the first = reads
// a file, as input parsing does; an @ later in a literal value does not.
func isFileAssignment(toComplete string) bool {
	_, value, found := strings.Cut(toComplete, "=")
	return found && strings.HasPrefix(value, "@")
}

func CompleteSetValues(
	resolveVersion ConnectorVersionResolver,
	paths PathCompleter,
) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if isFileAssignment(toComplete) || resolveVersion == nil {
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
	if isFileAssignment(toComplete) {
		marker := strings.Index(toComplete, "=@")
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
	if key, prefix, found := strings.Cut(toComplete, "="); found {
		return completeEnumValues(fields[key], prefix)
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
	return candidates, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveKeepOrder
}

// schemaFieldDescription is one line: required marker, first sentence and
// allowed values. connectors show prints the full field list.
func schemaFieldDescription(field configuration.SchemaField) string {
	description := fctl.FirstSentence(field.Description)
	if description == "" {
		description = "environment configuration"
		if field.Kind == configuration.ConfigFile {
			description = "file configuration"
		}
	}
	if len(field.Enum) > 0 {
		description += " [" + strings.Join(field.Enum, " | ") + "]"
	}
	if field.Required {
		description = "required · " + description
	}
	return description
}

// completeEnumValues offers KEY=value candidates for a field with an enum.
func completeEnumValues(field configuration.SchemaField, prefix string) ([]string, cobra.ShellCompDirective) {
	candidates := make([]string, 0, len(field.Enum))
	for _, value := range field.Enum {
		if strings.HasPrefix(value, prefix) {
			candidates = append(candidates, field.Key+"="+value)
		}
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
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
