package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func newPluginsCommand(settings *connection.Settings) *cobra.Command {
	root := &cobra.Command{Use: "plugins", Short: "Prepare and inspect external service plugins"}
	root.AddCommand(newPluginsListCommand(settings), newPluginsShowCommand(settings), newPluginsSyncCommand(settings), newPluginsInstallCommand(settings))
	return root
}

func pluginManager(settings *connection.Settings, cmd *cobra.Command) (*pluginmanager.Manager, error) {
	dir, err := settings.DirectoryPath()
	if err != nil {
		return nil, err
	}
	client := settings.HTTPClient(cmd.ErrOrStderr())
	client.CheckRedirect = nil // The manager validates public download redirects.
	return pluginmanager.New(filepath.Join(dir, "plugins"), client)
}

func writePluginResult(cmd *cobra.Command, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return command.WriteJSON(cmd.OutOrStdout(), data)
}

func pluginSyncTarget(ctx context.Context, settings *connection.Settings, cmd *cobra.Command, version string) (pluginmanager.Target, string, error) {
	return pluginServiceSyncTarget(ctx, settings, cmd, "ledger", version)
}

func pluginServiceSyncTarget(ctx context.Context, settings *connection.Settings, cmd *cobra.Command, service, version string) (pluginmanager.Target, string, error) {
	if _, err := supportedPluginService(service); err != nil {
		return pluginmanager.Target{}, "", err
	}
	if version == "" {
		client, err := settings.Client(ctx, cmd, service)
		if err != nil {
			return pluginmanager.Target{}, "", err
		}
		version, err = pluginServiceVersion(ctx, client, service)
		if err != nil {
			return pluginmanager.Target{}, "", err
		}
	}
	target, err := pluginServiceTarget(settings, cmd, service)
	if err != nil {
		return target, "", err
	}
	options, _, _, _, err := settings.Resolve(cmd)
	if err != nil {
		return target, "", err
	}
	if err := connection.Validate(options); err != nil {
		return target, "", err
	}
	if options.AuthMode == "cloud" && (target.Organization == "" || target.Stack == "") {
		err = fmt.Errorf("choose --organization and --stack, or omit --service-version to select the Cloud target")
	}
	return target, strings.TrimPrefix(version, "v"), err
}

type pluginSummary struct {
	Profile      string `json:"profile"`
	Organization string `json:"organization,omitzero"`
	Stack        string `json:"stack,omitzero"`
	Endpoint     string `json:"endpoint"`
	Service      string `json:"service"`
	Version      string `json:"version"`
	Revision     int    `json:"revision"`
	Platform     string `json:"platform"`
	Source       string `json:"source"`
	SHA256       string `json:"sha256"`
}

func summarizePlugin(lock pluginmanager.Lock) pluginSummary {
	source := lock.ArtifactDigest
	if lock.Local {
		source = "local"
	}
	return pluginSummary{Profile: lock.Target.Profile, Organization: lock.Target.Organization, Stack: lock.Target.Stack,
		Endpoint: lock.Target.Endpoint, Service: lock.Service, Version: lock.ServiceVersion, Revision: lock.Revision,
		Platform: lock.Platform.String(), Source: source, SHA256: lock.SHA256}
}

// External providers may replace only the root and connection boundary of their service.
func ledgerManifestIdentity(manifest pluginsdk.Manifest) error {
	return serviceManifestIdentity("ledger", manifest)
}

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
