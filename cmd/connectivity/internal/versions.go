package internal

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	xmodsemver "golang.org/x/mod/semver"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const shortDigestLength = 19 // "sha256:" plus 12 hex characters

// NormalizeSemver adds the "v" prefix golang.org/x/mod/semver expects.
func NormalizeSemver(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

// CompareSemver orders versions by SemVer precedence, then lexically so
// distinct spellings of equal precedence stay deterministic.
func CompareSemver(left, right string) int {
	if result := xmodsemver.Compare(NormalizeSemver(left), NormalizeSemver(right)); result != 0 {
		return result
	}
	return strings.Compare(left, right)
}

// CompleteConnectorVersions lists a connector's published versions newest
// first, marking the latest and shortening digests. The shell keeps this order
// instead of sorting lexically, which would put alpha.10 before alpha.2.
func CompleteConnectorVersions(ctx context.Context, client *sdk.FormanceV1, connector, toComplete string) ([]string, cobra.ShellCompDirective) {
	if connector == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	versions, err := CollectPagesBounded(CompletionPageSize, CompletionMaxPages,
		func(options ListOptions) ([]connectivity.ConnectorVersionSummary, bool, string, error) {
			response, err := client.ListConnectorVersions(ctx, operations.ListConnectorVersionsRequest{Name: url.PathEscape(connector), Cursor: OptionalString(options.Cursor), PageSize: fctl.Ptr(int(options.PageSize)), Query: OptionalString(options.Query)}, nil)
			var page *connectivity.ConnectorVersionCursor
			if response != nil {
				page = response.ConnectorVersionCursor
			}
			if err != nil || page == nil {
				return nil, false, "", err
			}
			return page.Cursor.Data, page.Cursor.HasMore, fctl.StringPointerToString(page.Cursor.Next), nil
		})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	sort.SliceStable(versions, func(i, j int) bool { return CompareSemver(versions[i].Version, versions[j].Version) > 0 })
	candidates := make([]string, 0, len(versions))
	for index, version := range versions {
		if version.Version == "" || !strings.HasPrefix(version.Version, toComplete) {
			continue
		}
		candidate := version.Version
		if description := VersionDescription(version, index == 0); description != "" {
			candidate += "\t" + description
		}
		candidates = append(candidates, candidate)
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// VersionDescription describes a version in completion: latest marker,
// release date and shortened digest.
func VersionDescription(version connectivity.ConnectorVersionSummary, latest bool) string {
	parts := make([]string, 0, 3)
	if latest {
		parts = append(parts, "latest")
	}
	if version.ReleaseDate != nil {
		parts = append(parts, "released "+version.ReleaseDate.Format(time.DateOnly))
	}
	if digest := fctl.StringPointerToString(version.Digest); digest != "" {
		parts = append(parts, fctl.Truncate(digest, shortDigestLength+1))
	}
	return strings.Join(parts, " · ")
}
