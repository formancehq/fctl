package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func executeSecrets(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := validID(req.Args[0]); err != nil {
		return nil, err
	}
	path := httpclient.Path("clients", req.Args[0], "secrets")
	if req.CommandPath[3] == "delete" {
		if err := validID(req.Args[1]); err != nil {
			return nil, err
		}
		return client.Do(ctx, http.MethodDelete, httpclient.Path("clients", req.Args[0], "secrets", req.Args[1]), nil, nil, nil)
	}
	fields, err := bodyObject(req)
	if err != nil {
		return nil, err
	}
	if err := applyName(fields, req, 1); err != nil {
		return nil, err
	}
	if err := validateFields(fields, true, true); err != nil {
		return nil, err
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, http.MethodPost, path, nil, body, nil)
}
