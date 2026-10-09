package plugins

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
)

// NewCommand builds the external plugin management commands.
func NewCommand(settings *connection.Settings) *cobra.Command {
	root := &cobra.Command{Use: "plugins", Short: "Prepare and inspect external service plugins"}
	root.AddCommand(newListCommand(settings), newShowCommand(settings), newSyncCommand(settings), newInstallCommand(settings))
	return root
}
