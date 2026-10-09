package cmd

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
)

func newPluginsShowCommand(settings *connection.Settings) *cobra.Command {
	show := &cobra.Command{Use: "show", Short: "Show the Ledger plugin locked for the selected target", Args: cobra.NoArgs}
	show.RunE = func(cmd *cobra.Command, _ []string) error {
		manager, err := pluginManager(settings, cmd)
		if err != nil {
			return err
		}
		target, err := ledgerTarget(settings, cmd)
		if err != nil {
			return err
		}
		lock, err := manager.Load(target, "ledger")
		if err != nil {
			return err
		}
		return writePluginResult(cmd, summarizePlugin(lock))
	}
	return show
}
