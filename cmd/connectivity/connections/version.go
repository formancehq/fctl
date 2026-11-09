package connections

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	xmodsemver "golang.org/x/mod/semver"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const (
	versionResolvePageSize = int32(100)
	versionResolveMaxPages = 100
)

// resolveConnectorVersion returns the ConnectorVersion whose configSchema governs
// the configuration being assembled: the requested pin (or channel alias) when
// there is one, the server-resolved stable head otherwise. This mirrors an
// unselected install, whose API write persists the stable selector.
func resolveConnectorVersion(ctx context.Context, client *sdk.FormanceV1, connector, pinned string) (*connectivity.ConnectorVersion, error) {
	if connector == "" {
		return nil, fmt.Errorf("connector name is required")
	}
	if pinned == "" {
		pinned = "stable"
	}
	response, err := client.GetConnectorVersion(ctx, operations.GetConnectorVersionRequest{Name: url.PathEscape(connector), Version: url.PathEscape(pinned)}, nil)
	var version *connectivity.ConnectorVersion
	if response != nil {
		version = response.ConnectorVersion
	}
	if err != nil {
		return nil, err
	}
	if version == nil || version.Version == "" {
		return nil, fmt.Errorf("get connectivity connector %q version %q: empty response", connector, pinned)
	}
	return version, nil
}

// The connector is immutable; the last applied version is the channel floor.
func appliedChannelFloor(instance *connectivity.Connection) string {
	if instance == nil || instance.ConnectionStatus == nil {
		return ""
	}
	return fctl.StringPointerToString(instance.ConnectionStatus.AppliedVersion)
}

// resolveChannelVersion mirrors the operator resolver for the schema selected
// during a channel-tracked configure. It reads every version page because the
// selected channel head is not necessarily present on the first page.
func resolveChannelVersion(ctx context.Context, client *sdk.FormanceV1, connector, channel, running string) (*connectivity.ConnectorVersion, error) {
	versions, err := connectivityinternal.CollectPages(versionResolvePageSize, versionResolveMaxPages,
		func(options connectivityinternal.ListOptions) ([]connectivity.ConnectorVersionSummary, bool, string, error) {
			response, err := client.ListConnectorVersions(ctx, operations.ListConnectorVersionsRequest{Name: url.PathEscape(connector), Cursor: connectivityinternal.OptionalString(options.Cursor), PageSize: fctl.Ptr(int(options.PageSize)), Query: connectivityinternal.OptionalString(options.Query)}, nil)
			var page *connectivity.ConnectorVersionCursor
			if response != nil {
				page = response.ConnectorVersionCursor
			}
			if err != nil {
				return nil, false, "", err
			}
			if page == nil {
				return nil, false, "", fmt.Errorf("list connectivity connector %q versions: empty response", connector)
			}
			// A missing data array is a malformed page, not an empty catalogue:
			// accepting it could select a version from an incomplete list.
			if page.Cursor.Data == nil {
				return nil, false, "", fmt.Errorf("list connectivity connector %q versions: cursor.data must be an array", connector)
			}
			return page.Cursor.Data, page.Cursor.HasMore, fctl.StringPointerToString(page.Cursor.Next), nil
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
		running = ""
	}
	seen := make(map[string]struct{})
	chosen := ""
	// The API rejects duplicate exact SemVer strings before channel filtering.
	// Distinct spellings with equal precedence use the server's lexical tie-break.
	for _, version := range versions {
		if !isStrictSemver(version) {
			continue
		}
		if _, exists := seen[version]; exists {
			return "", fmt.Errorf("multiple ConnectorVersions found for version %q", version)
		}
		seen[version] = struct{}{}
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
	return connectivityinternal.NormalizeSemver(version)
}

func compareSemver(left, right string) int {
	return connectivityinternal.CompareSemver(left, right)
}
