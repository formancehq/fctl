package plugins

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
)

func newShowCommand(settings *connection.Settings) *cobra.Command {
	var service string
	show := &cobra.Command{Use: "show", Short: "Show the service plugin locked for the selected target", Args: cobra.NoArgs}
	show.Flags().StringVar(&service, "service", "ledger", "Service plugin: ledger or auth")
	show.RunE = func(cmd *cobra.Command, _ []string) error {
		lock, err := pluginhost.Show(settings, cmd, service)
		if err != nil {
			return err
		}
		return writePluginResult(cmd, summarizePlugin(lock))
	}
	return show
}
