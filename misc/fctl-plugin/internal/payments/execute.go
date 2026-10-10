package payments

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func paymentsOperation(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op commandapi.Operation, body json.RawMessage) (json.RawMessage, error) {
	major, minor, err := paymentsVersion(ctx, client)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(op.Command, "connectors ") {
		return paymentConnector(ctx, client, r, op, body, major)
	}
	v3, err := paymentV3Route(op.Command, major, minor)
	if err != nil {
		return nil, err
	}
	if err := validatePaymentPayload(op.Command, body, major, minor); err != nil {
		return nil, err
	}
	if v3 {
		op.Segments = append([]string{"v3"}, op.Segments...)
	}
	query, err := commandapi.QueryValues(op.Query, r)
	if err != nil {
		return nil, err
	}
	if op.Command == "pools balances" {
		query.Set("at", r.Args[1])
	}
	if cursor := query.Get("cursor"); cursor != "" {
		query = url.Values{"cursor": {cursor}}
		body = nil
	}
	path, err := commandapi.Route(op.Segments, r)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, op.Method, path, query, body, nil)
}
