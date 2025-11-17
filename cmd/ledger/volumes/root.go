package volumes

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
)

func NewLedgerVolumesCommand() *cobra.Command {
	return fctl.NewCommand("volumes",
		fctl.WithAliases("vol", "volume", "vols", "vlm"),
		fctl.WithShortDescription("Get volumes and Balances for accounts"),
		fctl.WithChildCommands(NewListCommand()),
	)

}
