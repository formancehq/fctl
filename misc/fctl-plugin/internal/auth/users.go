package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func executeUsers(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	path := httpclient.Path("users")
	if req.CommandPath[len(req.CommandPath)-1] == "show" {
		if err := validID(req.Args[0]); err != nil {
			return nil, err
		}
		path = httpclient.Path("users", req.Args[0])
	}
	return client.Do(ctx, http.MethodGet, path, nil, nil, nil)
}
