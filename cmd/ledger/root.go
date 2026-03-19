package ledger

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/ledger/accounts"
	"github.com/formancehq/fctl/v3/cmd/ledger/internal"
	"github.com/formancehq/fctl/v3/cmd/ledger/transactions"
	"github.com/formancehq/fctl/v3/cmd/ledger/volumes"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewStackCommand("ledger",
		fctl.WithAliases("l"),
		fctl.WithPersistentStringFlag(internal.LedgerFlag, "default", "Specific ledger"),
		fctl.WithShortDescription("Ledger management"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewSendCommand(),
			NewStatsCommand(),
			NewServerInfoCommand(),
			NewListCommand(),
			NewSetMetadataCommand(),
			NewDeleteMetadataCommand(),
			NewExportCommand(),
			NewImportCommand(),
			transactions.NewLedgerTransactionsCommand(),
			accounts.NewLedgerAccountsCommand(),
			volumes.NewLedgerVolumesCommand(),
		),
	)
}
