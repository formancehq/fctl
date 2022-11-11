package me

import (
	"github.com/formancehq/fctl/cmd/cloud/me/invitations"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
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
