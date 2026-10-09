package plugins

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
)

func newListCommand(settings *connection.Settings) *cobra.Command {
	list := &cobra.Command{Use: "list", Short: "List plugins prepared for your profiles and targets", Args: cobra.NoArgs}
	list.RunE = func(cmd *cobra.Command, _ []string) error {
		locks, err := pluginhost.List(settings, cmd)
		if err != nil {
			return err
		}
		rows := make([]pluginSummary, 0, len(locks))
		for _, lock := range locks {
			rows = append(rows, summarizePlugin(lock))
		}
		return writePluginResult(cmd, rows)
	}
	return list
}
