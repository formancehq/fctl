package me

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/subcmds/cloud/me/invitations"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdbuilder.NewCommand("me",
		cmdbuilder.WithChildCommands(
			invitations.NewCommand(),
			NewInfoCommand(),
		),
	)
}
