package cmd

import (
	"github.com/spf13/cobra"
)

func newOrganizationsCommand() *cobra.Command {
	return newStackCommand("organizations",
		withChildCommands(
			newOrganizationsListCommand(),
		),
	)
}
