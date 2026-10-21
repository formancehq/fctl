package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func prepareAccounts(ctx context.Context, client *httpclient.Client, name string, r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodGet, query: make(url.Values)}
	switch name {
	case "accounts list":
		body, err := filterPayload(r.Flags["metadata"], "")
		if err != nil {
			return op, err
		}
		op.path, op.body = httpclient.Path("v2", r.Flags["ledger"], "accounts"), body
		if err := paginate(op.query, r); err != nil {
			return op, err
		}
	case "accounts show":
		if err := validateSegment(r.Args[0], "account address"); err != nil {
			return op, err
		}
		op.path = httpclient.Path(r.Flags["ledger"], "accounts", r.Args[0])
	case "accounts set-metadata", "accounts delete-metadata":
		return metadataOperation(ctx, client, "accounts", r)
	default:
		return op, fmt.Errorf("unknown ledger command %q", name)
	}
	return op, nil
}

func volumesList(r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodGet, path: httpclient.Path("v2", r.Flags["ledger"], "volumes"), query: make(url.Values)}
	var err error
	op.body, err = filterPayload(r.Flags["metadata"], r.Flags["address"])
	if err != nil {
		return op, err
	}
	if err := paginate(op.query, r); err != nil {
		return op, err
	}
	if r.Flags["cursor"] != "" {
		op.query.Set("pageSize", r.Flags["page-size"]) // Historical volumes controller always supplies both.
	}
	op.query.Set("groupBy", r.Flags["group-by"])
	op.query.Set("insertionDate", r.Flags["insertion-date"])
	if err := dateQuery(op.query, r.Flags, map[string]string{"start-time": "startTime", "end-time": "endTime"}); err != nil {
		return op, err
	}
	return op, nil
}

func filterPayload(metadata, address string) (json.RawMessage, error) {
	values, err := flagPairs(metadata)
	if err != nil {
		return nil, err
	}
	// Stable ordering keeps fixtures and request fingerprints reproducible.
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	clauses := make([]map[string]any, 0, len(keys)+1)
	for _, key := range keys {
		clauses = append(clauses, map[string]any{"$match": map[string]string{"metadata[" + key + "]": values[key]}})
	}
	if address != "" {
		clauses = append(clauses, map[string]any{"$match": map[string]string{"account": address}})
	}
	return json.Marshal(map[string]any{"$and": clauses})
}
