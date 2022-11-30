package profiles

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return internal.NewCommand("show",
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithAliases("s"),
		internal.WithShortDescription("Show profile"),
		internal.WithValidArgsFunction(ProfileNamesAutoCompletion),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {

			config, err := internal.Get(cmd)
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
