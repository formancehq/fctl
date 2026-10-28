package internal

import (
	"fmt"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

// CollectPages drains a cursor-paginated list endpoint, following
// `cursor.next` until the server reports the last page. A user-facing read
// must never render an unmarked prefix, so malformed and truncated cursors are
// reported as errors.
func CollectPages[T any](pageSize int32, maxPages int, fetch func(connectivityclient.ListOptions) ([]T, bool, string, error)) ([]T, error) {
	return collectPages(pageSize, maxPages, false, fetch)
}

// CollectPagesBounded is for best-effort shell completion. It returns the
// useful prefix at the intentional page bound, while retaining validation of
// malformed cursors before that bound.
func CollectPagesBounded[T any](pageSize int32, maxPages int, fetch func(connectivityclient.ListOptions) ([]T, bool, string, error)) ([]T, error) {
	return collectPages(pageSize, maxPages, true, fetch)
}

func collectPages[T any](pageSize int32, maxPages int, allowTruncation bool, fetch func(connectivityclient.ListOptions) ([]T, bool, string, error)) ([]T, error) {
	if maxPages <= 0 {
		return nil, fmt.Errorf("pagination maximum pages must be positive")
	}
	options := connectivityclient.ListOptions{PageSize: pageSize}
	items := make([]T, 0)
	seen := make(map[string]struct{})
	for page := 0; page < maxPages; page++ {
		pageItems, hasMore, next, err := fetch(options)
		if err != nil {
			return nil, err
		}
		items = append(items, pageItems...)
		if !hasMore {
			return items, nil
		}
		if next == "" {
			return nil, fmt.Errorf("pagination cursor hasMore is true without a next cursor")
		}
		if _, repeated := seen[next]; repeated {
			return nil, fmt.Errorf("pagination cursor repeated %q", next)
		}
		seen[next] = struct{}{}
		if page+1 == maxPages {
			if allowTruncation {
				return items, nil
			}
			return nil, fmt.Errorf("pagination exceeded maximum of %d pages", maxPages)
		}
		options.Cursor = next
	}
	return items, nil // unreachable while maxPages is positive
}
