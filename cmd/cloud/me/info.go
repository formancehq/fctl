package me

import (
	"errors"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewInfoCommand() *cobra.Command {
	return cmdbuilder.NewCommand("info",
		cmdbuilder.WithAliases("i", "in"),
		cmdbuilder.WithShortDescription("Display user information"),
		cmdbuilder.WithArgs(cobra.ExactArgs(0)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}

			profile := config.GetCurrentProfile(cmd, cfg)
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
