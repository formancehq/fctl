package me

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewInfoCommand() *cobra.Command {
	return cmdbuilder.NewCommand("info",
		cmdbuilder.WithAliases("i"),
		cmdbuilder.WithShortDescription("Display user information"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			profile := config.GetCurrentProfile(cmd.Context(), cfg)

			relyingParty, err := membership.GetRelyingParty(cmd.Context(), profile)
			if err != nil {
				return err
			}

			userInfo, err := profile.GetUserInfo(cmd.Context(), relyingParty)
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
