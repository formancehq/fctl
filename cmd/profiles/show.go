package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithAliases("s"),
		cmdbuilder.WithShortDescription("Show profile"),
		cmdbuilder.WithValidArgsFunction(ProfileNamesAutoCompletion),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			config, err := config.Get(cmd)
			if err != nil {
				return err
			}

			p := config.GetProfile(args[0])
			if p == nil {
				return errors.New("not found")
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("Membership URI"), p.GetMembershipURI()})
			tableData = append(tableData, []string{pterm.LightCyan("Default organization"), p.GetDefaultOrganization()})
			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
