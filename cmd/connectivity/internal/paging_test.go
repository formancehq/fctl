package internal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

func TestCollectPagesRejectsDishonestCursorEnvelopes(t *testing.T) {
	tests := map[string]struct {
		maxPages int
		fetch    func(connectivityclient.ListOptions) ([]string, bool, string, error)
		want     string
	}{
		"has more without next cursor": {
			maxPages: 2,
			fetch: func(connectivityclient.ListOptions) ([]string, bool, string, error) {
				return []string{"one"}, true, "", nil
			},
			want: "hasMore",
		},
		"repeated cursor": {
			maxPages: 3,
			fetch: func(options connectivityclient.ListOptions) ([]string, bool, string, error) {
				return []string{options.Cursor}, true, "again", nil
			},
			want: "repeated",
		},
		"bounded before final page": {
			maxPages: 1,
			fetch: func(connectivityclient.ListOptions) ([]string, bool, string, error) {
				return []string{"one"}, true, "next", nil
			},
			want: "maximum",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := CollectPages(100, tt.maxPages, tt.fetch)

			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tt.want), "error = %v", err)
		})
	}
}

func TestCollectPagesBoundedReturnsUsefulCompletionPrefix(t *testing.T) {
	items, err := CollectPagesBounded(100, 1, func(options connectivityclient.ListOptions) ([]string, bool, string, error) {
		return []string{options.Cursor + "candidate"}, true, "later", nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"candidate"}, items)
}
