package profiles

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal.NewCommand("list",
		internal.WithAliases("l"),
		internal.WithShortDescription("List profiles"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			currentProfileName := internal.GetCurrentProfileName(cmd, cfg)

			profiles := internal.MapKeys(cfg.GetProfiles())
			tableData := internal.Map(profiles, func(p string) []string {
				isCurrent := "No"
				if p == currentProfileName {
					isCurrent = "Yes"
				}
				return []string{p, isCurrent}
			})
			tableData = internal.Prepend(tableData, []string{"Name", "Active"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}))
}
