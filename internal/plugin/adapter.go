package plugin

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/interactive"
)

// Resolver supplies the selected service endpoint and authenticated HTTP client.
type Resolver func(context.Context, string) (*api.Client, error)

// RequestResolver can select a named profile using already validated flags.
type RequestResolver func(context.Context, string, pluginsdk.ExecuteRequest) (*api.Client, error)

// Adapter installs commands from a frozen registry into the host root.
type Adapter struct {
	registry       *Registry
	resolve        Resolver
	resolveRequest RequestResolver
}

func NewCommand(registry *Registry, resolver Resolver) *Adapter {
	return &Adapter{registry: registry, resolve: resolver}
}

func NewCommandWithRequest(registry *Registry, resolver RequestResolver) *Adapter {
	return &Adapter{registry: registry, resolveRequest: resolver}
}

// AddTo validates every tree before attaching any command. Help and completion
// use only manifest metadata; runtime client resolution happens inside RunE.
func (a *Adapter) AddTo(root *cobra.Command) error {
	if root == nil || a.registry == nil || (a.resolve == nil && a.resolveRequest == nil) {
		return fmt.Errorf("plugin adapter requires root, registry and resolver")
	}
	var commands []*cobra.Command
	for _, entry := range a.registry.snapshot() {
		if err := checkHostCollisions(root, entry.manifest.Root); err != nil {
			return err
		}
		for _, existing := range root.Commands() {
			if existing.Name() == pluginsdk.CommandName(entry.manifest.Root) || existing.HasAlias(pluginsdk.CommandName(entry.manifest.Root)) {
				return fmt.Errorf("plugin root collides with host command %q", existing.Name())
			}
		}
		tree, err := a.build(entry, entry.manifest.Root, nil, nil, "")
		if err != nil {
			return err
		}
		commands = append(commands, tree)
	}
	root.AddCommand(commands...)
	return nil
}

func checkHostCollisions(root *cobra.Command, spec pluginsdk.CommandSpec) error {
	for _, flag := range spec.Flags {
		if root.PersistentFlags().Lookup(flag.Name) != nil || root.InheritedFlags().Lookup(flag.Name) != nil {
			return fmt.Errorf("plugin flag %q collides with host", flag.Name)
		}
		if flag.Shorthand != "" && (root.PersistentFlags().ShorthandLookup(flag.Shorthand) != nil || root.InheritedFlags().ShorthandLookup(flag.Shorthand) != nil) {
			return fmt.Errorf("plugin shorthand %q collides with host", flag.Shorthand)
		}
	}
	for _, child := range spec.Subcommands {
		if err := checkHostCollisions(root, child); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) build(entry registration, spec pluginsdk.CommandSpec, parentPath []string, inherited []pluginsdk.FlagSpec, target string) (*cobra.Command, error) {
	path := append(append([]string{}, parentPath...), pluginsdk.CommandName(spec))
	flags := append(append([]pluginsdk.FlagSpec{}, inherited...), spec.Flags...)
	if spec.Target != "" {
		target = spec.Target
	}
	cmd := &cobra.Command{Use: spec.Use, Aliases: spec.Aliases, Short: spec.Short, Long: spec.Long, Example: spec.Example, Annotations: map[string]string{"fctl.target": target}}
	cmd.Args = commandArgs(spec)
	for _, flag := range spec.Flags {
		if canSupplyFlag(spec.Inputs, flag) || canSupplyFileBody(spec.Files, flag) {
			flag.Required = false // The SDK still enforces this after collection.
		}
		if err := addFlag(cmd, flag); err != nil {
			return nil, err
		}
	}
	if spec.Runnable {
		cmd.RunE = func(cmd *cobra.Command, args []string) error { return a.run(cmd, entry, path, flags, args) }
	}
	next := append([]pluginsdk.FlagSpec{}, inherited...)
	for _, flag := range spec.Flags {
		if flag.Persistent {
			next = append(next, flag)
		}
	}
	for _, child := range spec.Subcommands {
		tree, err := a.build(entry, child, path, next, target)
		if err != nil {
			return nil, err
		}
		cmd.AddCommand(tree)
	}
	return cmd, nil
}

func addFlag(cmd *cobra.Command, flag pluginsdk.FlagSpec) error {
	set := cmd.Flags()
	if flag.Persistent {
		set = cmd.PersistentFlags()
	}
	switch flag.Type {
	case "string":
		set.StringP(flag.Name, flag.Shorthand, flag.Default, flag.Usage)
	case "bool":
		value, err := strconv.ParseBool(flag.Default)
		if err != nil {
			return err
		}
		set.BoolP(flag.Name, flag.Shorthand, value, flag.Usage)
	case "uint32":
		value, err := strconv.ParseUint(flag.Default, 10, 32)
		if err != nil {
			return err
		}
		set.Uint32P(flag.Name, flag.Shorthand, uint32(value), flag.Usage)
	default:
		return fmt.Errorf("unsupported flag type %q", flag.Type)
	}
	if flag.Required {
		if flag.Persistent {
			return cmd.MarkPersistentFlagRequired(flag.Name)
		}
		return cmd.MarkFlagRequired(flag.Name)
	}
	return nil
}

func (a *Adapter) run(cmd *cobra.Command, entry registration, path []string, flags []pluginsdk.FlagSpec, args []string) error {
	if interactive.Enabled(cmd) {
		return a.runInteractive(cmd, entry, path, flags, args)
	}
	normalized, err := prepareRequest(cmd, entry.manifest, path, flags, args)
	if err != nil {
		return err
	}
	service, err := pluginsdk.CommandService(entry.manifest, path)
	if err != nil {
		return err
	}
	client, err := a.resolveClient(cmd.Context(), service, normalized)
	if err != nil {
		return err
	}
	if client == nil || client.HTTPClient() == nil {
		return fmt.Errorf("plugin resolver returned no HTTP client")
	}
	normalized.Endpoint = client.Endpoint()
	normalized.Context = client.Context()
	instance := entry.factory(client.HTTPClient())
	if instance == nil {
		return fmt.Errorf("plugin factory returned nil")
	}
	response, execErr := instance.Execute(cmd.Context(), normalized)
	spec, err := pluginsdk.FindCommand(entry.manifest, path)
	if err != nil {
		return err
	}
	return renderPluginOutput(cmd, spec.Files, normalized, response, execErr)
}

func (a *Adapter) resolveClient(ctx context.Context, service string, request pluginsdk.ExecuteRequest) (*api.Client, error) {
	if a.resolveRequest != nil {
		return a.resolveRequest(ctx, service, request)
	}
	return a.resolve(ctx, service)
}

func prepareRequest(cmd *cobra.Command, manifest pluginsdk.Manifest, path []string, flags []pluginsdk.FlagSpec, args []string) (pluginsdk.ExecuteRequest, error) {
	normalized, err := normalizeRequest(cmd, manifest, path, flags, args)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	spec, err := pluginsdk.FindCommand(manifest, path)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	return readPluginRequest(cmd, manifest, spec.Files, normalized, flags)
}

// Normalize the request before opening a body or plugin input file. A declared
// file source can satisfy a required body flag once the host has read it.
func normalizeRequest(cmd *cobra.Command, manifest pluginsdk.Manifest, path []string, flags []pluginsdk.FlagSpec, args []string) (pluginsdk.ExecuteRequest, error) {
	request, err := requestFromFlags(cmd, path, flags, args)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	spec, err := pluginsdk.FindCommand(manifest, path)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	source, err := pluginInputSource(spec.Files, request)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	if source != "" {
		manifest = fileInputManifest(manifest)
	}
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	if err := checkPluginInput(spec.Files, normalized, flags); err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	if _, err := pluginOutputFormat(spec.Files, normalized); err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	if _, err := pluginOutputDestination(spec.Files, normalized); err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	return normalized, nil
}

func fileInputManifest(manifest pluginsdk.Manifest) pluginsdk.Manifest {
	manifest = cloneManifest(manifest)
	var relax func(*pluginsdk.CommandSpec)
	relax = func(spec *pluginsdk.CommandSpec) {
		for i := range spec.Flags {
			if spec.Flags[i].Body {
				spec.Flags[i].Required = false
			}
		}
		for i := range spec.Subcommands {
			relax(&spec.Subcommands[i])
		}
	}
	relax(&manifest.Root)
	return manifest
}

func readPluginRequest(cmd *cobra.Command, manifest pluginsdk.Manifest, files *pluginsdk.FileSpec, request pluginsdk.ExecuteRequest, flags []pluginsdk.FlagSpec) (pluginsdk.ExecuteRequest, error) {
	if err := checkPluginInput(files, request, flags); err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	if _, err := pluginOutputFormat(files, request); err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	if _, err := pluginOutputDestination(files, request); err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	request, err := readRequestBody(cmd, request, flags)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	request, err = ReadPluginInput(cmd, files, request)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	return pluginsdk.NormalizeRequest(manifest, request)
}

func requestFromFlags(cmd *cobra.Command, path []string, flags []pluginsdk.FlagSpec, args []string) (pluginsdk.ExecuteRequest, error) {
	request := pluginsdk.ExecuteRequest{CommandPath: append([]string{}, path...), Args: append([]string{}, args...), Flags: map[string]string{}, ChangedFlags: map[string]bool{}}
	for _, spec := range flags {
		flag := lookupFlag(cmd, spec.Name)
		if flag == nil {
			return pluginsdk.ExecuteRequest{}, fmt.Errorf("plugin flag %q is missing", spec.Name)
		}
		request.Flags[spec.Name] = flag.Value.String()
		request.ChangedFlags[spec.Name] = flag.Changed
	}
	return request, nil
}

func readRequestBody(cmd *cobra.Command, normalized pluginsdk.ExecuteRequest, flags []pluginsdk.FlagSpec) (pluginsdk.ExecuteRequest, error) {
	var err error
	for _, spec := range flags {
		if spec.Body && (normalized.Flags[spec.Name] != "" || normalized.ChangedFlags[spec.Name]) {
			normalized.Body, err = command.ReadBody(cmd, normalized.Flags[spec.Name])
			if err != nil {
				return pluginsdk.ExecuteRequest{}, err
			}
		}
	}
	return normalized, nil
}

func lookupFlag(cmd *cobra.Command, name string) *pflag.Flag {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return flag
	}
	return cmd.InheritedFlags().Lookup(name)
}

func commandArgs(spec pluginsdk.CommandSpec) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if interactive.Enabled(cmd) && len(spec.Inputs) != 0 {
			return cobra.MaximumNArgs(spec.Args.Max)(cmd, args)
		}
		return cobra.RangeArgs(spec.Args.Min, spec.Args.Max)(cmd, args)
	}
}
