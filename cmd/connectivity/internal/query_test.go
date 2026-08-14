package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildListQueryReturnsEmptyWithoutInput(t *testing.T) {
	query, err := BuildListQuery("", nil)

	require.NoError(t, err)
	require.Empty(t, query)
}

func TestBuildListQueryBuildsMatchLeafFromEquality(t *testing.T) {
	query, err := BuildListQuery("", []string{"catalog=ee"})

	require.NoError(t, err)
	require.JSONEq(t, `{"$match":{"catalog":"ee"}}`, query)
}

func TestBuildListQueryConjoinsMultipleFilters(t *testing.T) {
	query, err := BuildListQuery("", []string{"catalog=ee", "phase=Ready"})

	require.NoError(t, err)
	require.JSONEq(t, `{"$and":[{"$match":{"catalog":"ee"}},{"$match":{"phase":"Ready"}}]}`, query)
}

// A bare $not also selects objects missing the key, so the builder conjoins
// $exists with the negation (the openapi documents this trap).
func TestBuildListQueryBuildsGuardedNegation(t *testing.T) {
	query, err := BuildListQuery("", []string{"phase!=Rejected"})

	require.NoError(t, err)
	require.JSONEq(t, `{"$and":[{"$exists":{"phase":true}},{"$not":{"$match":{"phase":"Rejected"}}}]}`, query)
}

func TestBuildListQueryBuildsLikeLeaf(t *testing.T) {
	query, err := BuildListQuery("", []string{"name~stripe%"})

	require.NoError(t, err)
	require.JSONEq(t, `{"$like":{"name":"stripe%"}}`, query)
}

func TestBuildListQueryPassesRawQueryThrough(t *testing.T) {
	raw := `{"$or":[{"$match":{"catalog":"ee"}},{"$exists":{"catalog":false}}]}`

	query, err := BuildListQuery(raw, nil)

	require.NoError(t, err)
	require.JSONEq(t, raw, query)
}

func TestBuildListQueryRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		filters []string
	}{
		{name: "raw query combined with filters", raw: `{"$match":{"catalog":"ee"}}`, filters: []string{"phase=Ready"}},
		{name: "raw query is not JSON", raw: `{$match:`},
		{name: "raw query is not an object", raw: `["$match"]`},
		{name: "filter without operator", filters: []string{"catalog"}},
		{name: "filter without key", filters: []string{"=ee"}},
		{name: "filter without value", filters: []string{"catalog="}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildListQuery(tt.raw, tt.filters)

			require.Error(t, err)
		})
	}
}
