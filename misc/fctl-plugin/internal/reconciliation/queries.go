package reconciliation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

// Continuation cursors replace the query and body, as in the historical CLI.
func paginatedList(name, use, path string, args int) command.Operation {
	op := command.Leaf(name, use, http.MethodGet, path, args, args, command.PaginationFlags()...)
	op.Query = map[string]string{"cursor": "cursor", "page-size": "pageSize"}
	op.Run = cursorOperation
	return op
}

func clarityList(group string) command.Operation {
	op := paginatedList(group+" list", "list", group, 0)
	op.Spec.Flags = append(op.Spec.Flags, command.StringFlag("query", "JSON query-builder expression"), command.DataFlag())
	op.Body = clarityListBody
	return op
}

func cursorOperation(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op command.Operation, body json.RawMessage) (json.RawMessage, error) {
	query, err := command.QueryValues(op.Query, r)
	if err != nil {
		return nil, err
	}
	if cursor := query.Get("cursor"); cursor != "" {
		query = url.Values{"cursor": {cursor}}
		body = nil
	}
	path, err := command.Route(op.Segments, r)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, op.Method, path, query, body, nil)
}

func clarityListBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Flags["cursor"] != "" {
		return nil, nil
	}
	if r.Body != nil && r.Flags["query"] != "" {
		return nil, fmt.Errorf("provide --query or --data once")
	}
	if r.Body != nil {
		return command.ObjectBody(r.Body)
	}
	if r.Flags["query"] != "" {
		return command.ObjectBody(json.RawMessage(r.Flags["query"]))
	}
	return nil, nil
}
