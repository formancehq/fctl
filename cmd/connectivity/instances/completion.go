package instances

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

const (
	instanceCompletionLimit   = int32(500)
	instanceCompletionTimeout = 2 * time.Second
)

type PathCompleter func(prefix string) ([]string, error)

func CompleteInstanceNames(factory connectivityinternal.ClientFactory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, completionCommand, cancel, ok := completionClient(cmd, factory)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		response, err := client.ListInstances(completionCommand.Context(), connectivityclient.ListOptions{Limit: instanceCompletionLimit})
		if err != nil || response == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		candidates := make([]string, 0, len(response.Items))
		for _, instance := range response.Items {
			name := stringValue(instance.Metadata.Name)
			if name == "" || !strings.HasPrefix(name, toComplete) {
				continue
			}
			description := strings.Join(nonEmptyStrings(instance.Spec.Plugin, instance.Spec.Ledger), " · ")
			if description != "" {
				name += "\t" + description
			}
			candidates = append(candidates, name)
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
}

func CompleteVersions(factory connectivityinternal.ClientFactory, pluginArg func(*cobra.Command, []string) string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if pluginArg == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		pluginName := pluginArg(cmd, args)
		if pluginName == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		client, completionCommand, cancel, ok := completionClient(cmd, factory)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()

		plugin, err := client.GetPlugin(completionCommand.Context(), pluginName)
		if err != nil || plugin == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		candidates := make([]string, 0, len(plugin.Spec.Versions))
		for _, version := range plugin.Spec.Versions {
			if version.Version == "" || !strings.HasPrefix(version.Version, toComplete) {
				continue
			}
			candidate := version.Version
			description := stringValue(version.Image)
			if description == "" {
				description = stringValue(version.Digest)
			}
			if description != "" {
				candidate += "\t" + description
			}
			candidates = append(candidates, candidate)
		}
		sort.Strings(candidates)
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
}

func CompleteSetValues(
	factory connectivityinternal.ClientFactory,
	resolvePlugin func(context.Context, connectivityclient.Client, *cobra.Command, []string) (*connectivityclient.Plugin, error),
	paths PathCompleter,
) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

		if resolvePlugin == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		client, completionCommand, cancel, ok := completionClient(cmd, factory)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cancel()
		plugin, err := resolvePlugin(completionCommand.Context(), client, completionCommand, args)
		if err != nil || plugin == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		fields, err := SchemaFields(plugin)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		supplied := suppliedSetKeys(cmd, args)
		ordered := make([]SchemaField, 0, len(fields))
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
}

func schemaFieldDescription(field SchemaField) string {
	if field.Description != "" {
		return field.Description
	}
	if field.Kind == ConfigFile {
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

func completionClient(cmd *cobra.Command, factory connectivityinternal.ClientFactory) (connectivityclient.Client, *cobra.Command, context.CancelFunc, bool) {
	if factory == nil {
		return nil, nil, func() {}, false
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, instanceCompletionTimeout)
	completionCommand := *cmd
	completionCommand.SetContext(connectivityinternal.WithNonInteractive(ctx))
	client, err := factory(&completionCommand)
	if err != nil {
		cancel()
		return nil, nil, func() {}, false
	}
	return client, &completionCommand, cancel, true
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
