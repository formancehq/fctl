package cmd

import (
	"github.com/formancehq/auth/authclient"
	fctl "github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	return newStackCommand("auth",
		withChildCommands(
			newAuthClientsCommand(),
		),
	)
}

func newAuthClient(cmd *cobra.Command) (*authclient.APIClient, error) {
	profile, err := getCurrentProfile()
	if err != nil {
		return nil, err
	}
	organizationID, err := resolveOrganizationID(cmd)
	if err != nil {
		return nil, err
	}
	stackID, err := resolveStackID(cmd, organizationID)
	if err != nil {
		return nil, err
	}
	return fctl.NewAuthClientFromContext(cmd.Context(), profile, getHttpClient(),
		organizationID, stackID)
}
