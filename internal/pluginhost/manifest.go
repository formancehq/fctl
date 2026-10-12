package pluginhost

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/plugin"
)

// External providers may replace only the root and connection boundary of their service.
func serviceManifestIdentity(service string, manifest pluginsdk.Manifest) error {
	descriptor, err := supportedPluginService(service)
	if err != nil {
		return err
	}
	if manifest.Name != descriptor.name || manifest.Service != descriptor.name || pluginsdk.CommandName(manifest.Root) != descriptor.name {
		return fmt.Errorf("%s plugins must use the %s service and command root", descriptor.title, descriptor.name)
	}
	return validateServiceOverrides(service, manifest.Root)
}

func validateServiceOverrides(service string, command pluginsdk.CommandSpec) error {
	if command.Service != "" && command.Service != service {
		return fmt.Errorf("%s plugin command %q cannot override service with %q", service, command.Use, command.Service)
	}
	for _, child := range command.Subcommands {
		if err := validateServiceOverrides(service, child); err != nil {
			return err
		}
	}
	return nil
}

//nolint:contextcheck // Validation builds callbacks without executing them.
func validateServiceManifest(ctx context.Context, root *cobra.Command, service string, manifest pluginsdk.Manifest) error {
	if err := serviceManifestIdentity(service, manifest); err != nil {
		return err
	}
	registry := &plugin.Registry{}
	factory := plugin.ExternalFactory("", manifest)
	if err := registry.Register(ctx, factory(nil), factory); err != nil {
		return fmt.Errorf("invalid %s plugin manifest: %w", service, err)
	}
	check := &cobra.Command{Use: "fctl"}
	check.PersistentFlags().AddFlagSet(root.PersistentFlags())
	return plugin.NewCommand(registry, func(context.Context, string) (*api.Client, error) {
		return nil, fmt.Errorf("manifest validation does not execute commands")
	}).AddTo(check)
}

func readPluginManifest(ctx context.Context, path string) (pluginsdk.Manifest, error) {
	instance, err := transport.Open(ctx, path, nil, "")
	if err != nil {
		return pluginsdk.Manifest{}, err
	}
	manifest, err := instance.GetManifest(ctx)
	return manifest, errors.Join(err, instance.Close())
}
