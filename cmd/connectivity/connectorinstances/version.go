package connectorinstances

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	xmodsemver "golang.org/x/mod/semver"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

const (
	versionResolvePageSize = int32(100)
	versionResolveMaxPages = 100
)

// resolveConnectorVersion returns the ConnectorVersion whose configSchema governs
// the configuration being assembled: the requested pin (or channel alias) when
// there is one, the server-resolved stable head otherwise. This mirrors an
// unselected install, whose API write persists the stable selector.
func resolveConnectorVersion(ctx context.Context, client connectivityclient.Client, connector, pinned string) (*connectivityclient.ConnectorVersion, error) {
	if connector == "" {
		return nil, fmt.Errorf("connector name is required")
	}
	if pinned == "" {
		pinned = "stable"
	}
	version, err := client.GetConnectorVersion(ctx, connector, pinned)
	var apiErr *connectivityclient.APIError
	if (pinned == "stable" || pinned == connectivityclient.VersionAliasLatest) && errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
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

// appliedChannelFloor returns the version currently applied to this exact
// connector. The API only uses this status value as a channel resolver floor
// after confirming its resolved connector identity; desired spec.version is
// deliberately not a floor when switching from a pin to a channel.
func appliedChannelFloor(instance *connectivityclient.ConnectorInstance) string {
	if instance == nil || instance.Status == nil || instance.Status.ResolvedVersion == nil {
		return ""
	}
	// Older API responses did not stamp resolvedConnectorRef. The server treats
	// that empty identity as the instance's previous connector, while a nonempty
	// different identity must never constrain this connector's channel.
	if instance.Status.ResolvedConnectorRef != nil && *instance.Status.ResolvedConnectorRef != "" && *instance.Status.ResolvedConnectorRef != instance.Spec.Connector {
		return ""
	}
	return *instance.Status.ResolvedVersion
}

// resolveChannelVersion mirrors the operator resolver for the schema selected
// during a channel-tracked configure. It reads every version page because the
// selected channel head is not necessarily present on the first page.
func resolveChannelVersion(ctx context.Context, client connectivityclient.Client, connector, channel, running string) (*connectivityclient.ConnectorVersion, error) {
	versions, err := connectivityinternal.CollectPages(versionResolvePageSize, versionResolveMaxPages,
		func(options connectivityclient.ListOptions) ([]connectivityclient.ConnectorVersionSummary, bool, string, error) {
			page, err := client.ListConnectorVersions(ctx, connector, options)
			if err != nil {
				return nil, false, "", err
			}
			if page == nil {
				return nil, false, "", fmt.Errorf("list connectivity connector %q versions: empty response", connector)
			}
			return page.Items, page.HasMore, page.Next, nil
		})
	if err != nil {
		return nil, err
	}
	candidates := make([]string, 0, len(versions))
	for _, version := range versions {
		candidates = append(candidates, version.Version)
	}
	selected, err := selectChannelVersion(candidates, channel, running)
	if err != nil {
		return nil, fmt.Errorf("resolve connectivity connector %q channel %q: %w", connector, channel, err)
	}
	return resolveConnectorVersion(ctx, client, connector, selected)
}

func selectChannelVersion(versions []string, channel, running string) (string, error) {
	allowed, ok := channelPrereleases[channel]
	if !ok {
		return "", fmt.Errorf("unsupported channel %q", channel)
	}
	if running != "" && !isStrictSemver(running) {
		return "", fmt.Errorf("parsing running version %q: invalid semantic version", running)
	}
	chosen := ""
	for _, version := range versions {
		if !isStrictSemver(version) {
			return "", fmt.Errorf("parsing candidate version %q: invalid semantic version", version)
		}
		if !channelAdmits(version, allowed) || (running != "" && (xmodsemver.Major(normalizeSemver(version)) != xmodsemver.Major(normalizeSemver(running)) || compareSemver(version, running) < 0)) {
			continue
		}
		if chosen == "" || compareSemver(version, chosen) > 0 {
			chosen = version
		}
	}
	if chosen == "" {
		return "", fmt.Errorf("no candidate version satisfies channel %q", channel)
	}
	return chosen, nil
}

var channelPrereleases = map[string]map[string]struct{}{
	"stable": {},
	"rc":     {"rc": {}},
	"beta":   {"rc": {}, "beta": {}},
	"alpha":  {"rc": {}, "beta": {}, "alpha": {}},
}

func channelAdmits(version string, allowed map[string]struct{}) bool {
	prerelease := strings.TrimPrefix(xmodsemver.Prerelease(normalizeSemver(version)), "-")
	if prerelease == "" {
		return true
	}
	identifier, _, _ := strings.Cut(prerelease, ".")
	_, ok := allowed[identifier]
	return ok
}

func isStrictSemver(version string) bool {
	core := strings.TrimPrefix(version, "v")
	if suffix := strings.IndexAny(core, "-+"); suffix >= 0 {
		core = core[:suffix]
	}
	return len(strings.Split(core, ".")) == 3 && xmodsemver.IsValid(normalizeSemver(version))
}

func normalizeSemver(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func compareSemver(left, right string) int {
	if result := xmodsemver.Compare(normalizeSemver(left), normalizeSemver(right)); result != 0 {
		return result
	}
	return strings.Compare(left, right)
}
