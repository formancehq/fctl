package plugin

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

// Resolver supplies the selected service endpoint and authenticated HTTP client.
type Resolver func(context.Context, string) (*api.Client, error)

// Adapter installs commands from a frozen registry into the host root.
type Adapter struct {
	registry *Registry
	resolve  Resolver
}

func NewCommand(registry *Registry, resolver Resolver) *Adapter {
	return &Adapter{registry: registry, resolve: resolver}
}

// AddTo validates every tree before attaching any command. Help and completion
// use only manifest metadata; runtime client resolution happens inside RunE.
func (a *Adapter) AddTo(root *cobra.Command) error {
	if root == nil || a.registry == nil || a.resolve == nil {
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
		tree, err := a.build(entry, entry.manifest.Root, nil, nil)
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

func (a *Adapter) build(entry registration, spec pluginsdk.CommandSpec, parentPath []string, inherited []pluginsdk.FlagSpec) (*cobra.Command, error) {
	path := append(append([]string{}, parentPath...), pluginsdk.CommandName(spec))
	flags := append(append([]pluginsdk.FlagSpec{}, inherited...), spec.Flags...)
	cmd := &cobra.Command{Use: spec.Use, Short: spec.Short, Long: spec.Long, Example: spec.Example, Args: cobra.RangeArgs(spec.Args.Min, spec.Args.Max)}
	for _, flag := range spec.Flags {
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
		tree, err := a.build(entry, child, path, next)
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
	normalized, err := prepareRequest(cmd, entry.manifest, path, flags, args)
	if err != nil {
		return err
	}
	client, err := a.resolve(cmd.Context(), entry.manifest.Service)
	if err != nil {
		return err
	}
	if client == nil || client.HTTPClient() == nil {
		return fmt.Errorf("plugin resolver returned no HTTP client")
	}
	normalized.Endpoint = client.Endpoint()
	instance := entry.factory(client.HTTPClient())
	if instance == nil {
		return fmt.Errorf("plugin factory returned nil")
	}
	response, execErr := instance.Execute(cmd.Context(), normalized)
	if len(response.Data) == 0 {
		return execErr
	}
	return errors.Join(execErr, command.WriteJSON(cmd.OutOrStdout(), response.Data))
}

func prepareRequest(cmd *cobra.Command, manifest pluginsdk.Manifest, path []string, flags []pluginsdk.FlagSpec, args []string) (pluginsdk.ExecuteRequest, error) {
	request := pluginsdk.ExecuteRequest{CommandPath: append([]string{}, path...), Args: append([]string{}, args...), Flags: map[string]string{}, ChangedFlags: map[string]bool{}}
	for _, spec := range flags {
		flag := lookupFlag(cmd, spec.Name)
		if flag == nil {
			return pluginsdk.ExecuteRequest{}, fmt.Errorf("plugin flag %q is missing", spec.Name)
		}
		request.Flags[spec.Name] = flag.Value.String()
		request.ChangedFlags[spec.Name] = flag.Changed
	}
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
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
