package me

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/cloud/me/invitations"
	fctl "github.com/formancehq/fctl/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("me",
		fctl.WithShortDescription("Current user management"),
		fctl.WithChildCommands(
			invitations.NewCommand(),
			NewInfoCommand(),
		),
	)
}
