package me

import (
	"github.com/formancehq/fctl/cmd/cloud/me/invitations"
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return internal.NewCommand("me",
		internal.WithShortDescription("Current use management"),
		internal.WithChildCommands(
			invitations.NewCommand(),
			NewInfoCommand(),
		),
	)
}
