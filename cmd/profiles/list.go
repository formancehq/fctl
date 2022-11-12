package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("l"),
		cmdbuilder.WithShortDescription("List profiles"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			currentProfileName := config.GetCurrentProfileName(cmd.Context(), cfg)

			profiles := collections.MapKeys(cfg.GetProfiles())
			tableData := collections.Map(profiles, func(p string) []string {
				isCurrent := "No"
				if p == currentProfileName {
					isCurrent = "Yes"
				}
				return []string{p, isCurrent}
			})
			tableData = collections.Prepend(tableData, []string{"Name", "Active"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}))
}
