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
	var binary, localVersion string
	install := &cobra.Command{
		Use: "install", Short: "Install a Ledger plugin executable built locally", Args: cobra.NoArgs,
		Annotations: map[string]string{"fctl.target": "stack"},
		Example:     "  fctl plugins install --binary ./build/fctl-plugin-ledger --profile local",
	}
	install.Flags().StringVar(&binary, "binary", "", "Explicit path to the trusted executable")
	install.Flags().StringVar(&localVersion, "service-version", "", "Exact service version; defaults to Ledger /_info discovery")
	install.RunE = func(cmd *cobra.Command, _ []string) error {
		return installLedgerPlugin(cmd, settings, binary, localVersion)
	}

	return install
}

func installLedgerPlugin(cmd *cobra.Command, settings *connection.Settings, binary, localVersion string) error {
	if binary == "" {
		return fmt.Errorf("provide --binary PATH to the Ledger plugin executable")
	}
	path, err := filepath.Abs(binary)
	if err != nil {
		return err
	}
	manifest, err := readPluginManifest(cmd.Context(), path)
	if err != nil {
		return err
	}

	if err := validateLedgerManifest(cmd.Context(), cmd.Root(), manifest); err != nil {
		return err
	}
	target, version, err := pluginSyncTarget(cmd.Context(), settings, cmd, localVersion)
	if err != nil {
		return err
	}
	if manifest.Version != version {
		return fmt.Errorf("plugin targets Ledger %s, selected service is %s; build the matching plugin", manifest.Version, version)
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
