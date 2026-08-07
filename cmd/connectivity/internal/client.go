package internal

import (
	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ClientFactory func(*cobra.Command) (connectivityclient.Client, error)

func NewClientFactory() ClientFactory {
	return func(cmd *cobra.Command) (connectivityclient.Client, error) {
		_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
		if err != nil {
			return nil, err
		}

		clients, err := fctl.NewStackClientsFromFlags(
			cmd,
			relyingParty,
			fctl.NewPTermDialog(),
			profileName,
			*profile,
		)
		if err != nil {
			return nil, err
		}

		return connectivityclient.New(clients.URI, clients.HTTPClient), nil
	}
}
