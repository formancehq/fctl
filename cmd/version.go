package cmd

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	Version   = "develop"
	BuildDate = "-"
	Commit    = "-"
)

func NewVersionCommand() *cobra.Command {
	return internal.NewCommand("version",
		internal.WithShortDescription("Get version"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("Version"), Version})
			tableData = append(tableData, []string{pterm.LightCyan("Date"), BuildDate})
			tableData = append(tableData, []string{pterm.LightCyan("Commit"), Commit})
			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
