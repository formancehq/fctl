package schemas

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewLedgerSchemasCommand() *cobra.Command {
	return fctl.NewCommand("schemas",
		fctl.WithAliases("schema", "sc"),
		fctl.WithShortDescription("Manage ledger schemas"),
		fctl.WithChildCommands(
			NewInsertCommand(),
			NewGetCommand(),
			NewListCommand(),
		),
	)
}
