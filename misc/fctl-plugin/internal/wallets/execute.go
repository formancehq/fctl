package wallets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func walletOperation(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op commandapi.Operation, body json.RawMessage) (json.RawMessage, error) {
	var err error
	if op.Command != "list" {
		r, err = resolveWalletTarget(ctx, client, r)
		if err != nil {
			return nil, err
		}
	}
	if op.Command == "credit" || op.Command == "debit" {
		body, err = walletSubjectBody(ctx, client, r, body)
		if err != nil {
			return nil, err
		}
	}
	query, err := walletQuery(op, r)
	if err != nil {
		return nil, err
	}
	path, err := commandapi.Route(op.Segments, r)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	if r.Flags["ik"] != "" {
		headers.Set("Idempotency-Key", r.Flags["ik"])
	}
	return client.Do(ctx, op.Method, path, query, body, headers)
}

func walletQuery(op commandapi.Operation, r pluginsdk.ExecuteRequest) (url.Values, error) {
	query, err := commandapi.QueryValues(op.Query, r)
	if err != nil {
		return nil, err
	}
	if cursor := query.Get("cursor"); cursor != "" {
		return url.Values{"cursor": {cursor}}, nil
	}
	if op.Method == http.MethodGet && r.Flags["metadata"] != "" {
		metadata, err := commandapi.Pairs(r.Flags["metadata"])
		if err != nil {
			return nil, err
		}
		for key, value := range metadata {
			query.Set("metadata["+key+"]", value)
		}
	}
	return query, nil
}
