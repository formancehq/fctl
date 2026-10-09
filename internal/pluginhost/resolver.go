package pluginhost

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
)

// RequestResolver resolves authenticated clients for plugin requests.
func RequestResolver(root *cobra.Command, settings *connection.Settings) plugin.RequestResolver {
	return func(ctx context.Context, service string, request pluginsdk.ExecuteRequest) (*api.Client, error) {
		cmd, _, err := root.Find(request.CommandPath)
		if err != nil {
			return nil, err
		}
		for _, name := range []string{"organization", "stack"} {
			if value := request.Context[name]; value != "" {
				if err := root.PersistentFlags().Set(name, value); err != nil {
					return nil, err
				}
			}
		}
		if service == "cloud-apps" {
			return settings.ApplicationClient(ctx, cmd, request.Flags["deploy-app-alias"])
		}
		return settings.Client(ctx, cmd, service)
	}
}
