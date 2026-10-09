package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	"github.com/formancehq/fctl/v4/internal/connection"
)

func newPluginsInstallCommand(settings *connection.Settings) *cobra.Command {
	var binary, localVersion, service string
	install := &cobra.Command{
		Use: "install", Short: "Install a service plugin executable built locally", Args: cobra.NoArgs,
		Annotations: map[string]string{"fctl.target": "stack"},
		Example:     "  fctl plugins install --binary ./build/fctl-plugin-ledger --profile local",
	}
	install.Flags().StringVar(&service, "service", "ledger", "Service plugin: ledger or auth")
	install.Flags().StringVar(&binary, "binary", "", "Explicit path to the trusted executable")
	install.Flags().StringVar(&localVersion, "service-version", "", "Exact service version; defaults to selected service /_info discovery")
	install.RunE = func(cmd *cobra.Command, _ []string) error {
		return installServicePlugin(cmd, settings, service, binary, localVersion)
	}

	return install
}

func installLedgerPlugin(cmd *cobra.Command, settings *connection.Settings, binary, localVersion string) error {
	return installServicePlugin(cmd, settings, "ledger", binary, localVersion)
}

func installServicePlugin(cmd *cobra.Command, settings *connection.Settings, service, binary, localVersion string) error {
	if err := distributionService(service); err != nil {
		return err
	}
	if binary == "" {
		return fmt.Errorf("provide --binary PATH to the %s plugin executable", service)
	}
	path, err := filepath.Abs(binary)
	if err != nil {
		return err
	}
	manifest, err := readPluginManifest(cmd.Context(), path)
	if err != nil {
		return err
	}

	if err := validateServiceManifest(cmd.Context(), cmd.Root(), service, manifest); err != nil {
		return err
	}
	target, version, err := pluginServiceSyncTarget(cmd.Context(), settings, cmd, service, localVersion)
	if err != nil {
		return err
	}
	if manifest.Version != version {
		return fmt.Errorf("plugin targets %s %s, selected service is %s; build the matching plugin", service, manifest.Version, version)
	}
	manager, err := pluginManager(settings, cmd)
	if err != nil {
		return err
	}
	lock, err := manager.InstallLocal(cmd.Context(), path, target, manifest)
	if err != nil {
		return err
	}
	return writePluginResult(cmd, summarizePlugin(lock))

}

func readPluginManifest(ctx context.Context, path string) (pluginsdk.Manifest, error) {
	instance, err := transport.Open(ctx, path, nil, "")
	if err != nil {
		return pluginsdk.Manifest{}, err
	}
	manifest, err := instance.GetManifest(ctx)
	return manifest, errors.Join(err, instance.Close())
}
