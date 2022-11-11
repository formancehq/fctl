package me

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
)

func NewInfoCommand() *cobra.Command {
	return cmdbuilder.NewCommand("info",
		cmdbuilder.WithShortDescription("Display user information"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get()
			if err != nil {
				return err
			}

			profile := config.GetCurrentProfile(cfg)

			relyingParty, err := membership.GetRelyingParty(profile)
			if err != nil {
				return err
			}

			userInfo, err := profile.GetUserInfo(cmd.Context(), relyingParty)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Subject: %s\r\n", userInfo.GetSubject())
			fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\r\n", userInfo.GetEmail())

			return nil
		}),
	)
}
