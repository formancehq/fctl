package factory

import (
	"context"
	"maps"

	"github.com/spf13/cobra"

	legacy "github.com/formancehq/fctl/misc/fctl-plugin"
	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
)

func serviceRequestResolver(root *cobra.Command, settings *connection.Settings, legacyRoots map[string]bool) plugin.RequestResolver {
	base := pluginhost.RequestResolver(root, settings)
	providers := maps.Clone(legacyRoots)
	return func(ctx context.Context, service string, request pluginsdk.ExecuteRequest) (*api.Client, error) {
		client, err := base(ctx, service, request)
		if err != nil {
			return nil, err
		}
		if !legacyRequest(providers, request) {
			return client, nil
		}
		line, err := settings.StackVersion(ctx, root)
		if err != nil {
			return nil, err
		}
		if line != "" && !legacy.LegacyStack(line) {
			return nil, legacy.Unsupported(service, "", line)
		}
		metadata := client.Context()
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["stack-version"] = line
		metadata["provider"] = "legacy"
		metadata["plugin-version"] = legacy.Version
		return client.WithContext(metadata), nil
	}
}

// Bind provider identity to this command tree. A concurrent cache update must
// never remove the Stack guard from an already registered legacy factory.
func legacyRequest(providers map[string]bool, request pluginsdk.ExecuteRequest) bool {
	return len(request.CommandPath) > 0 && providers[request.CommandPath[0]]
}
