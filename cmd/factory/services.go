// Package factory assembles the service providers used by the root command.
package factory

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
	cloudplugin "github.com/formancehq/fctl/v4/plugins/cloud"
)

// Services holds providers and preparation failures for a single command tree.
type Services struct {
	factories []plugin.Factory
	missing   []*cobra.Command
	errors    map[string]error
}

// Prepare selects service providers from the requested target and cached metadata.
func Prepare(ctx context.Context, root *cobra.Command, settings *connection.Settings, args []string) *Services {
	authFactory, authErr := pluginhost.PrepareService(ctx, root, settings, args, "auth")
	ledgerFactory, ledgerErr := pluginhost.PrepareService(ctx, root, settings, args, "ledger")
	services := &Services{
		factories: []plugin.Factory{cloudplugin.New},
		errors:    map[string]error{"auth": authErr, "ledger": ledgerErr},
	}
	for _, provider := range []struct {
		name, title string
		factory     plugin.Factory
		err         error
	}{
		{"ledger", "Ledger", ledgerFactory, ledgerErr},
		{"auth", "Auth", authFactory, authErr},
	} {
		if provider.factory != nil {
			services.factories = append(services.factories, provider.factory)
		} else {
			services.missing = append(services.missing, pluginhost.UnpreparedServiceCommand(provider.name, provider.title, provider.err))
		}
	}
	return services
}

// PreparationError returns the failure for the service selected by a command.
func (s *Services) PreparationError(cmd *cobra.Command) error {
	return s.errors[pluginhost.ServiceCommand(cmd)]
}

// AddTo registers providers and attaches service commands to the root.

func (s *Services) AddTo(root *cobra.Command, settings *connection.Settings) error {
	root.AddCommand(s.missing...)
	registry := &plugin.Registry{}
	for _, provider := range s.factories {
		if err := registry.Register(context.WithoutCancel(root.Context()), provider(nil), provider); err != nil {
			return err
		}
	}
	return plugin.NewCommandWithRequest(registry, pluginhost.RequestResolver(root, settings)).AddTo(root)
}
