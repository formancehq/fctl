package me

import (
	"errors"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewInfoCommand() *cobra.Command {
	return internal.NewCommand("info",
		internal.WithAliases("i", "in"),
		internal.WithShortDescription("Display user information"),
		internal.WithArgs(cobra.ExactArgs(0)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			profile := internal.GetCurrentProfile(cmd, cfg)
			if !profile.IsConnected() {
				return errors.New("Not logged. Use 'login' command before.")
			}

			userInfo, err := profile.GetUserInfo(cmd)
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("Subject"), userInfo.GetSubject()})
			tableData = append(tableData, []string{pterm.LightCyan("Email"), userInfo.GetEmail()})

			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
