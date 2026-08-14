package connectorinstances

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

// resolveConnectorVersion returns the ConnectorVersion whose configSchema governs
// the configuration being assembled: the requested pin (or channel alias) when
// there is one, the server-resolved `latest` alias otherwise.
func resolveConnectorVersion(ctx context.Context, client connectivityclient.Client, connector, pinned string) (*connectivityclient.ConnectorVersion, error) {
	if connector == "" {
		return nil, fmt.Errorf("connector name is required")
	}
	if pinned == "" {
		pinned = connectivityclient.VersionAliasLatest
	}
	version, err := client.GetConnectorVersion(ctx, connector, pinned)
	var apiErr *connectivityclient.APIError
	if pinned == connectivityclient.VersionAliasLatest && errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("connectivity connector %q has no published version: %w", connector, err)
	}
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
