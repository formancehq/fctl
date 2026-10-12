package command_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/testutil"
)

func fixturePlugin(client *http.Client) pluginsdk.Plugin {
	list := command.Leaf("list", "list", http.MethodGet, "/resources", 0, 0, command.PaginationFlags()...)
	list.Query = map[string]string{"page-size": "pageSize"}
	return command.New("fixture", "1.0.0", client, []command.Operation{
		list,
		command.Confirmed(command.Leaf("delete", "delete ID", http.MethodDelete, "/resources/$0", 1, 1)),
	}, nil)
}

func TestHTTPFailuresAndNumbers(t *testing.T) {
	t.Parallel()
	for _, status := range []int{400, 404, 409, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Parallel()
			failure := `{"errorCode":"VALIDATION","errorMessage":"service rejected request","data":{"amount":9007199254740993}}`
			server, verify := testutil.Fixture(t, []testutil.Exchange{{Method: "DELETE", Path: "/resources/id", Status: status, Response: failure}})
			defer verify()
			p := fixturePlugin(server.Client())
			response, err := p.Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"fixture", "delete"}, Args: []string{"id"}, Flags: map[string]string{"confirm": "true"}, Endpoint: server.URL + "/gateway/service"})
			var httpError *httpclient.Error
			if !errors.As(err, &httpError) || httpError.StatusCode != status || httpError.Code != "VALIDATION" {
				t.Fatalf("HTTP failure: %v", err)
			}
			if string(response.Data) != failure {
				t.Errorf("lost partial response: %s", response.Data)
			}
		})
	}
	server, verify := testutil.Fixture(t, []testutil.Exchange{{Method: "GET", Path: "/resources", Query: "pageSize=100", Response: `{"amount":9007199254740993}`}, {Method: "GET", Path: "/resources", Query: "pageSize=100", Response: `not json`}})
	defer verify()
	p := fixturePlugin(server.Client())
	r := pluginsdk.ExecuteRequest{CommandPath: []string{"fixture", "list"}, Endpoint: server.URL + "/gateway/service"}
	response, err := p.Execute(t.Context(), r)
	if err != nil || string(response.Data) != `{"amount":9007199254740993}` {
		t.Fatalf("raw response: %s, %v", response.Data, err)
	}
	if _, err := p.Execute(t.Context(), r); err == nil {
		t.Fatal("accepted malformed service JSON")
	}
}

func TestCancellationPropagates(t *testing.T) {
	t.Parallel()
	server, verify := testutil.Fixture(t, nil)
	defer verify()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := fixturePlugin(server.Client()).Execute(ctx, pluginsdk.ExecuteRequest{CommandPath: []string{"fixture", "list"}, Endpoint: server.URL + "/gateway/service"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestInvalidInputMakesNoRequests(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, want string
		request    pluginsdk.ExecuteRequest
	}{
		{"route traversal", "dot segment", pluginsdk.ExecuteRequest{CommandPath: []string{"fixture", "delete"}, Args: []string{".."}, Flags: map[string]string{"confirm": "true"}}},
		{"missing confirmation", "confirm", pluginsdk.ExecuteRequest{CommandPath: []string{"fixture", "delete"}, Args: []string{"id"}}},
		{"unexpected body", "JSON body", pluginsdk.ExecuteRequest{CommandPath: []string{"fixture", "list"}, Body: []byte(`{}`)}},
		{"invalid pagination", "page-size", pluginsdk.ExecuteRequest{CommandPath: []string{"fixture", "list"}, Flags: map[string]string{"page-size": "0"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, verify := testutil.Fixture(t, nil)
			defer verify()
			tc.request.Endpoint = server.URL + "/gateway/service"
			if _, err := fixturePlugin(server.Client()).Execute(t.Context(), tc.request); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}
