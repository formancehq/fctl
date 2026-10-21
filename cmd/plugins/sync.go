package plugins

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
)

func newSyncCommand(settings *connection.Settings) *cobra.Command {
	var catalogue, serviceVersion, service string
	var revision int
	sync := &cobra.Command{
		Use: "sync", Short: "Download the service plugin for the exact deployed service version", Args: cobra.NoArgs,
		Annotations: map[string]string{"fctl.target": "stack"},
		Example:     "  fctl plugins sync --organization ORG --stack STACK\n  fctl plugins sync --catalogue registry.yaml --service-version 3.0.0 --profile local",
	}
	sync.Flags().StringVar(&service, "service", "ledger", "Service plugin: ledger or auth")
	sync.Flags().StringVar(&catalogue, "catalogue", "", "YAML or JSON catalogue file or HTTPS URL; defaults to the official registry (FCTL_PLUGIN_CATALOGUE)")
	sync.Flags().StringVar(&serviceVersion, "service-version", "", "Exact service version; defaults to selected service /_info discovery")
	sync.Flags().IntVar(&revision, "revision", 0, "Plugin revision; defaults to the highest for this exact service version")
	sync.RunE = func(cmd *cobra.Command, _ []string) error {
		lock, err := pluginhost.Sync(cmd, settings, service, catalogue, serviceVersion, revision)
		if err != nil {
			return err
		}
		return writePluginResult(cmd, summarizePlugin(lock))
	}

	return sync
}
