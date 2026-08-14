package internal

import (
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

// CollectPages drains a cursor-paginated list endpoint, following
// `cursor.next` until the server reports the last page or maxPages is
// reached (a bound against pathological continuations).
func CollectPages[T any](pageSize int32, maxPages int, fetch func(connectivityclient.ListOptions) ([]T, bool, string, error)) ([]T, error) {
	options := connectivityclient.ListOptions{PageSize: pageSize}
	items := make([]T, 0)
	for page := 0; page < maxPages; page++ {
		pageItems, hasMore, next, err := fetch(options)
		if err != nil {
			return nil, err
		}
		items = append(items, pageItems...)
		if !hasMore || next == "" {
			break
		}
		options.Cursor = next
	}
	return items, nil
}
