package cmd

import (
	fctl "github.com/formancehq/fctl/cmd/internal"
	"github.com/numary/payments/client"
	"github.com/spf13/cobra"
)

func newPaymentsCommand() *cobra.Command {
	return newStackCommand("payments",
		withChildCommands(
			newPaymentsConnectorsCommand(),
		),
	)
}

func newPaymentsClient(cmd *cobra.Command) (*client.APIClient, error) {
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
	return fctl.NewPaymentsClientFromContext(cmd.Context(), profile, getHttpClient(),
		organizationID, stackID)
}
