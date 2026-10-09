package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func newPluginsSyncCommand(settings *connection.Settings) *cobra.Command {
	var catalogue, serviceVersion string
	var revision int
	sync := &cobra.Command{
		Use: "sync", Short: "Download the Ledger plugin for the exact deployed service version", Args: cobra.NoArgs,
		Annotations: map[string]string{"fctl.target": "stack"},
		Example:     "  fctl plugins sync --organization ORG --stack STACK\n  fctl plugins sync --catalogue registry.yaml --service-version 3.0.0 --profile local",
	}
	sync.Flags().StringVar(&catalogue, "catalogue", "", "YAML or JSON catalogue file or HTTPS URL; defaults to the official registry (FCTL_PLUGIN_CATALOGUE)")
	sync.Flags().StringVar(&serviceVersion, "service-version", "", "Exact service version; defaults to Ledger /_info discovery")
	sync.Flags().IntVar(&revision, "revision", 0, "Plugin revision; defaults to the highest for this exact service version")
	sync.RunE = func(cmd *cobra.Command, _ []string) error {
		return syncLedgerPlugin(cmd, settings, catalogue, serviceVersion, revision)
	}

	return sync
}

func syncLedgerPlugin(cmd *cobra.Command, settings *connection.Settings, catalogue, serviceVersion string, revision int) error {
	manager, err := pluginManager(settings, cmd)
	if err != nil {
		return err
	}
	catalogue, err = pluginCatalogue(cmd, settings, manager, catalogue)
	if err != nil {
		return err
	}

	target, version, err := pluginSyncTarget(cmd.Context(), settings, cmd, serviceVersion)
	if err != nil {
		return err
	}
	release, err := manager.Resolve(cmd.Context(), catalogue, "ledger", version, revision)
	if err != nil {
		return err
	}
	if err := validateLedgerManifest(cmd.Context(), cmd.Root(), release.Manifest); err != nil {
		return err
	}
	lock, err := manager.InstallResolved(cmd.Context(), catalogue, target, release)
	if err != nil {
		return err
	}
	return writePluginResult(cmd, summarizePlugin(lock))

}

func pluginCatalogue(cmd *cobra.Command, settings *connection.Settings, manager *pluginmanager.Manager, catalogue string) (string, error) {
	if catalogue != "" {
		return catalogue, nil
	}
	if value := os.Getenv("FCTL_PLUGIN_CATALOGUE"); value != "" {
		return value, nil
	}
	lock, err := cachedLedgerLock(settings, cmd, manager)
	if err == nil && lock.Catalogue != "" {
		return lock.Catalogue, nil
	}
	return pluginmanager.DefaultCatalogue, nil
}
