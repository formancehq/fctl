package plugins

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginselection"
)

func newSelectionsCommand(settings *connection.Settings) *cobra.Command {
	cmd := &cobra.Command{Use: "selections", Short: "List the legacy or modern provider selected for each service target", Args: cobra.NoArgs}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		directory, err := settings.DirectoryPath()
		if err != nil {
			return err
		}
		selections, err := pluginselection.List(filepath.Join(directory, "plugins"))
		if err != nil {
			return err
		}
		return writePluginResult(cmd, selections)
	}
	return cmd
}
