package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/formancehq/fctl/v3/pkg/client"
)

func TestCatalogAndTagCompletionHonourFiltersAlreadyTyped(t *testing.T) {
	var facetQueries, listQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/connectivity/_query/capabilities":
			_, _ = fmt.Fprint(w, `{"resources":{"connectors":{"catalog":{"operators":["$match"]},"tags":{"operators":["$match"]}}}}`)
		case "/api/connectivity/connectors/_facets":
			facetQueries = append(facetQueries, r.URL.Query().Get("query"))
			_, _ = fmt.Fprint(w, `{"total":9,"facets":{"region":{"europe":7,"africa":0}}}`)
		case "/api/connectivity/connectors":
			listQueries = append(listQueries, r.URL.Query().Get("query"))
			_, _ = fmt.Fprint(w, `{"cursor":{"pageSize":100,"hasMore":false,"data":[{"name":"ee-a","catalog":"ee"},{"name":"ee-b","catalog":"ee"},{"name":"oss-a","catalog":"oss"}]}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	api := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client())).Connectivity.V1
	existing := []string{"tags=provider:psp"}

	tags, _ := completeFilterExpressionsWithin(t.Context(), api, ResourceConnectors, "tags=region:", existing)
	catalogs, _ := completeFilterExpressionsWithin(t.Context(), api, ResourceConnectors, "catalog=", existing)

	require.Equal(t, []string{"tags=region:europe\t7 connectors"}, tags, "zero-count values are dropped")
	require.Equal(t, []string{"catalog=ee\t2 connectors", "catalog=oss\t1 connector"}, catalogs)
	require.Equal(t, []string{`{"$match":{"tags":"provider:psp"}}`}, facetQueries)
	require.Equal(t, []string{`{"$match":{"tags":"provider:psp"}}`}, listQueries)
}
