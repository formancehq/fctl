package internal

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	sdk "github.com/formancehq/fctl/v3/pkg/client"
)

func TestFilterCompletionWithSDK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/connectivity/_query/capabilities":
			if _, err := fmt.Fprint(w, `{"resources":{"connectors":{"name":{"operators":["$match"]},"tags":{"operators":["$match"]}},"connections":{"channel":{"operators":["$match"],"enum":["stable","beta"]},"connector":{"operators":["$match"]}}}}`); err != nil {
				t.Error(err)
			}
		case "/api/connectivity/connectors/_facets":
			if _, err := fmt.Fprint(w, `{"total":2,"facets":{"provider":{"stripe":2}}}`); err != nil {
				t.Error(err)
			}
		case "/api/connectivity/connectors":
			if r.URL.Query().Get("cursor") == "" {
				if _, err := fmt.Fprint(w, `{"cursor":{"pageSize":1,"hasMore":true,"next":"page2","data":[{"name":"stripe"}]}}`); err != nil {
					t.Error(err)
				}
			} else {
				if _, err := fmt.Fprint(w, `{"cursor":{"pageSize":1,"hasMore":false,"data":[{"name":"adyen"}]}}`); err != nil {
					t.Error(err)
				}
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	api := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client())).Connectivity.V1
	keys, directive := completeFilterExpressions(t.Context(), api, ResourceConnectors, "n")
	require.Equal(t, []string{"name=\toperators: $match"}, keys)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)
	tags, _ := completeFilterExpressions(t.Context(), api, ResourceConnectors, "tags=provider:")
	require.Equal(t, []string{"tags=provider:stripe\t2 connectors"}, tags)
	names, _ := completeFilterExpressions(t.Context(), api, ResourceConnections, "connector=")
	require.Equal(t, []string{"connector=adyen", "connector=stripe"}, names)
	channels, _ := completeChannels(t.Context(), api, "b")
	require.Equal(t, []string{"beta"}, channels)
}

func TestCompletionSDKFailureIsSilent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	api := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client())).Connectivity.V1
	candidates, directive := completeFilterExpressions(context.Background(), api, ResourceConnectors, "")
	require.Empty(t, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompletionWithoutProfileIsSilent(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config-dir", t.TempDir(), "")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	client, _, cancel, ok := CompletionClient(cmd)
	defer cancel()
	require.False(t, ok)
	require.Nil(t, client)
	require.Empty(t, out.String())
}
