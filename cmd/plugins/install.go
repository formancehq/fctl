package plugins

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
)

func newInstallCommand(settings *connection.Settings) *cobra.Command {
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
		lock, err := pluginhost.Install(cmd, settings, service, binary, localVersion)
		if err != nil {
			return err
		}
		return writePluginResult(cmd, summarizePlugin(lock))
	}

	return install
}
