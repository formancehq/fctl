package internal

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	oidcclient "github.com/formancehq/go-libs/v4/oidc/client"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ClientFactory func(*cobra.Command) (connectivityclient.Client, error)

type nonInteractiveContextKey struct{}

func WithNonInteractive(ctx context.Context) context.Context {
	return context.WithValue(ctx, nonInteractiveContextKey{}, true)
}

func IsNonInteractive(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	nonInteractive, _ := ctx.Value(nonInteractiveContextKey{}).(bool)
	return nonInteractive
}

type clientFactoryDependencies struct {
	loadAndAuthenticateCurrentProfile func(*cobra.Command) (*fctl.Config, *fctl.Profile, string, oidcclient.RelyingParty, error)
	resolveStackID                    func(*cobra.Command, fctl.Profile) (string, string, error)
	readStackToken                    func(*cobra.Command, string, string, string) (*fctl.AccessToken, error)
	newStackClientsFromFlags          func(*cobra.Command, oidcclient.RelyingParty, fctl.Dialog, string, fctl.Profile) (*fctl.StackClients, error)
}

func NewClientFactory() ClientFactory {
	return newClientFactory(clientFactoryDependencies{
		loadAndAuthenticateCurrentProfile: fctl.LoadAndAuthenticateCurrentProfile,
		resolveStackID:                    fctl.ResolveStackID,
		readStackToken:                    fctl.ReadStackToken,
		newStackClientsFromFlags:          fctl.NewStackClientsFromFlags,
	})
}

func newClientFactory(dependencies clientFactoryDependencies) ClientFactory {
	return func(cmd *cobra.Command) (connectivityclient.Client, error) {
		_, profile, profileName, relyingParty, err := dependencies.loadAndAuthenticateCurrentProfile(cmd)
		if err != nil {
			return nil, err
		}

		dialog := fctl.NewPTermDialog()
		if IsNonInteractive(cmd.Context()) {
			organizationID, stackID, err := dependencies.resolveStackID(cmd, *profile)
			if err != nil {
				return nil, err
			}
			stackToken, err := dependencies.readStackToken(cmd, profileName, organizationID, stackID)
			if err != nil {
				return nil, err
			}
			if stackToken == nil || stackToken.Expired() {
				return nil, fmt.Errorf("connectivity completion requires an existing unexpired stack token")
			}
			dialog = silentDialog{}
		}

		clients, err := dependencies.newStackClientsFromFlags(
			cmd,
			relyingParty,
			dialog,
			profileName,
			*profile,
		)
		if err != nil {
			return nil, err
		}

		return connectivityclient.New(clients.URI, clients.HTTPClient), nil
	}
}

type silentDialog struct{}

func (silentDialog) Info(string, ...any) {}
