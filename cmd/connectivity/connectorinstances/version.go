package connectorinstances

import (
	"context"
	"fmt"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

// resolveConnectorVersion returns the ConnectorVersion whose configSchema governs
// the configuration being assembled: the requested pin when there is one, the
// newest published version otherwise. The catalog serves versions ascending by
// semantic version, so the newest is the last item.
func resolveConnectorVersion(ctx context.Context, client connectivityclient.Client, connector, pinned string) (*connectivityclient.ConnectorVersion, error) {
	if connector == "" {
		return nil, fmt.Errorf("connector name is required")
	}
	if pinned == "" {
		versions, err := client.ListConnectorVersions(ctx, connector)
		if err != nil {
			return nil, err
		}
		if versions == nil || len(versions.Items) == 0 {
			return nil, fmt.Errorf("connectivity connector %q has no published version", connector)
		}
		pinned = versions.Items[len(versions.Items)-1].Version
	}
	version, err := client.GetConnectorVersion(ctx, connector, pinned)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, fmt.Errorf("get connectivity connector %q version %q: empty response", connector, pinned)
	}
	return version, nil
}

// instanceVersionPin reports the version an existing connector instance is
// configured against: its explicit pin when set, the version the operator
// actually applied otherwise. An empty result means "resolve the newest".
func instanceVersionPin(instance *connectivityclient.ConnectorInstance) string {
	if instance == nil {
		return ""
	}
	if instance.Spec.Version != nil && *instance.Spec.Version != "" {
		return *instance.Spec.Version
	}
	if instance.Status != nil && instance.Status.ResolvedVersion != nil {
		return *instance.Status.ResolvedVersion
	}
	return ""
}
