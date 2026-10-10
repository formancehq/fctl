// Package ledger ports cmd/ledger from fctl's historical main revision
// e00243b3e2e56aae6a09d7010b0c17890134388c onto the public plugin SDK.
package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

type plugin struct{ http *http.Client }

// New injects the host's HTTP transport. Manifest discovery works with nil.
// Endpoint is the service base URL, e.g. https://stack.example/api/ledger.
func New(client *http.Client) pluginsdk.Plugin { return &plugin{http: client} }

func (*plugin) GetManifest(context.Context) (pluginsdk.Manifest, error) { return manifest(), nil }

func (p *plugin) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	request, err := pluginsdk.NormalizeRequest(manifest(), request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if p.http == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("ledger execution requires an injected HTTP client")
	}
	client, err := httpclient.New(request.Endpoint, p.http)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	name := strings.Join(request.CommandPath[1:], " ")
	if request.Body != nil && name != "transactions num" && name != "schemas insert" && name != "import" {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("%s does not accept a JSON body", name)
	}
	if name == "import" {
		return importLogs(ctx, client, request)
	}
	if name == "export" {
		if err := validateSegment(request.Flags["ledger"], "ledger"); err != nil {
			return pluginsdk.ExecuteResponse{}, err
		}
		return exportLogs(ctx, client, request.Flags["ledger"])
	}
	op, err := prepare(ctx, client, name, request)
	if err != nil {
		return response(nil, err)
	}
	data, err := client.Do(ctx, op.method, op.path, op.query, op.body, nil)
	return response(data, err)
}

func response(data json.RawMessage, err error) (pluginsdk.ExecuteResponse, error) {
	if failure, ok := errors.AsType[*httpclient.Error](err); ok && failure.Body != nil {
		data = failure.Body
	}
	return pluginsdk.ExecuteResponse{Data: data}, err
}

type operation struct {
	method string
	path   string
	query  url.Values
	body   json.RawMessage
}

func prepare(ctx context.Context, client *httpclient.Client, name string, r pluginsdk.ExecuteRequest) (operation, error) {
	ledgerName := r.Flags["ledger"]
	if name != "create" && name != "list" && name != "server-infos" && name != "set-metadata" && name != "delete-metadata" {
		if err := validateSegment(ledgerName, "ledger"); err != nil {
			return operation{}, err
		}
	}
	op := operation{method: http.MethodGet, query: make(url.Values)}
	switch name {
	case "server-infos":
		op.path = "/_info"
	case "stats":
		op.path = httpclient.Path(ledgerName, "stats")
	case "list":
		op.path = "/v2"
		op.body = json.RawMessage("null") // The historical generated SDK serializes its nil filter map.
		op.query.Set("includeDeleted", "false")
		if err := paginate(op.query, r); err != nil {
			return op, err
		}
	case "create":
		return createOperation(r)
	case "set-metadata", "delete-metadata":
		return metadataOperation(ctx, client, "ledger", r)
	case "send":
		body, err := sendPayload(r)
		op.method, op.path, op.body = http.MethodPost, httpclient.Path(ledgerName, "transactions"), body
		return op, err
	case "accounts list":
		body, err := filterPayload(r.Flags["metadata"], "")
		if err != nil {
			return op, err
		}
		op.path, op.body = httpclient.Path("v2", ledgerName, "accounts"), body
		if err := paginate(op.query, r); err != nil {
			return op, err
		}
	case "accounts show":
		if err := validateSegment(r.Args[0], "account address"); err != nil {
			return op, err
		}
		op.path = httpclient.Path(ledgerName, "accounts", r.Args[0])
	case "accounts set-metadata", "accounts delete-metadata":
		return metadataOperation(ctx, client, "accounts", r)
	case "transactions list":
		return transactionList(r)
	case "transactions num":
		body, err := numPayload(r)
		op.method, op.path, op.body = http.MethodPost, httpclient.Path(ledgerName, "transactions"), body
		return op, err
	case "transactions show", "transactions revert":
		return transactionOperation(ctx, client, name, r)
	case "transactions set-metadata", "transactions delete-metadata":
		return metadataOperation(ctx, client, "transactions", r)
	case "schemas list":
		return schemasList(r)
	case "schemas get", "schemas insert":
		return schemaOperation(name, r)
	case "volumes list":
		return volumesList(r)
	default:
		return op, fmt.Errorf("unknown ledger command %q", name)
	}
	return op, nil
}

func schemasList(r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodGet, path: httpclient.Path("v2", r.Flags["ledger"], "schemas"), query: url.Values{"order": {"desc"}, "sort": {"created_at"}}}
	if err := paginate(op.query, r); err != nil {
		return op, err
	}
	if r.Flags["cursor"] != "" {
		// Historical generated SDK default even when the controller supplies only a cursor.
		op.query.Set("pageSize", "15")
	}
	return op, nil
}

func createOperation(r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodPost}
	if err := validateSegment(r.Args[0], "ledger name"); err != nil {
		return op, err
	}
	metadata, err := flagPairs(r.Flags["metadata"])
	if err != nil {
		return op, err
	}
	features, err := flagPairs(r.Flags["features"])
	if err != nil {
		return op, err
	}
	body := map[string]any{"bucket": r.Flags["bucket"]}
	if len(metadata) > 0 {
		body["metadata"] = metadata
	}
	if len(features) > 0 {
		body["features"] = features
	}
	op.path = httpclient.Path("v2", r.Args[0])
	op.body, err = json.Marshal(body)
	return op, err
}

func transactionOperation(ctx context.Context, client *httpclient.Client, name string, r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodGet, query: make(url.Values)}
	id, err := transactionID(ctx, client, r.Flags["ledger"], r.Args[0])
	if err != nil {
		return op, err
	}
	op.path = httpclient.Path(r.Flags["ledger"], "transactions", id)
	if name == "transactions show" {
		return op, nil
	}
	op.method = http.MethodPost
	op.path = httpclient.Path(r.Flags["ledger"], "transactions", id, "revert")
	if r.Flags["at-effective-date"] == "true" {
		op.path = httpclient.Path("v2", r.Flags["ledger"], "transactions", id, "revert")
		op.query.Set("atEffectiveDate", "true")
		op.query.Set("force", r.Flags["force"])
	} else {
		op.query.Set("disableChecks", r.Flags["force"])
	}
	return op, nil
}

func schemaOperation(name string, r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodGet}
	if err := validateSegment(r.Args[0], "schema version"); err != nil {
		return op, err
	}
	op.path = httpclient.Path("v2", r.Flags["ledger"], "schemas", r.Args[0])
	if name == "schemas insert" {
		if err := validateSchema(r.Body); err != nil {
			return op, err
		}
		op.method, op.body = http.MethodPost, r.Body
	} else if format := r.Flags["format"]; format != "json" && format != "yaml" && format != "yml" {
		return op, fmt.Errorf("unsupported format %q (expected json or yaml)", format)
	}
	return op, nil
}

func metadataOperation(ctx context.Context, client *httpclient.Client, resource string, r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodPost}
	isDelete := r.CommandPath[len(r.CommandPath)-1] == "delete-metadata"
	if isDelete {
		if err := validateSegment(r.Args[1], "metadata key"); err != nil {
			return op, err
		}
		op.method = http.MethodDelete
	} else {
		metadata, err := pairs(r.Args[1:])
		if err != nil {
			return op, err
		}
		op.body, err = json.Marshal(metadata)
		if err != nil {
			return op, err
		}
	}
	segments := []string{r.Flags["ledger"], resource, r.Args[0], "metadata"}
	if resource == "transactions" {
		id, err := transactionID(ctx, client, r.Flags["ledger"], r.Args[0])
		if err != nil {
			return op, err
		}
		segments[2] = id
	} else if err := validateSegment(r.Args[0], resource+" identifier"); err != nil {
		return op, err
	}
	if resource == "ledger" {
		segments = []string{"v2", r.Args[0], "metadata"}
		if !isDelete {
			op.method = http.MethodPut
		}
	} else if isDelete {
		segments = append([]string{"v2"}, segments...)
	}
	if isDelete {
		segments = append(segments, r.Args[1])
	}
	op.path = httpclient.Path(segments...)
	return op, nil
}

func transactionID(ctx context.Context, client *httpclient.Client, ledgerName, value string) (string, error) {
	if !strings.HasPrefix(value, "last") {
		id, err := nonnegativeInteger(value, "transaction ID")
		if err != nil {
			return "", err
		}
		return id.String(), nil
	}
	offset := int64(0)
	if suffix := strings.TrimPrefix(value, "last"); suffix != "" {
		var err error
		offset, err = strconv.ParseInt(suffix, 10, 64)
		if err != nil || offset < 0 {
			return "", fmt.Errorf("invalid transaction offset %q", value)
		}
	}
	data, err := client.Do(ctx, http.MethodGet, httpclient.Path(ledgerName, "transactions"), url.Values{"pageSize": {"1"}}, nil, nil)
	if err != nil {
		return "", err
	}
	var result struct {
		Cursor struct {
			Data []struct {
				ID json.RawMessage `json:"txid"`
			} `json:"data"`
		} `json:"cursor"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("decode last transaction: %w", err)
	}
	if len(result.Cursor.Data) == 0 {
		return "", fmt.Errorf("no transaction found")
	}
	id, err := nonnegativeInteger(string(result.Cursor.Data[0].ID), "last transaction ID")
	if err != nil {
		return "", err
	}
	id.Sub(id, big.NewInt(offset))
	if id.Sign() < 0 {
		return "", fmt.Errorf("transaction offset exceeds last transaction ID")
	}
	return id.String(), nil
}

func transactionList(r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodGet, path: httpclient.Path(r.Flags["ledger"], "transactions"), query: make(url.Values)}
	if err := paginate(op.query, r); err != nil {
		return op, err
	}
	metadata, err := flagPairs(r.Flags["metadata"])
	if err != nil {
		return op, err
	}
	if r.Flags["cursor"] != "" {
		return op, nil
	}
	for flag, query := range map[string]string{"account": "account", "src": "source", "dst": "destination", "reference": "reference"} {
		if value := r.Flags[flag]; value != "" {
			op.query.Set(query, value)
		}
	}
	for key, value := range metadata {
		op.query.Set("metadata["+key+"]", value)
	}
	if err := dateQuery(op.query, r.Flags, map[string]string{"start": "startTime", "end": "endTime"}); err != nil {
		return op, err
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

func paginate(query url.Values, r pluginsdk.ExecuteRequest) error {
	size, err := strconv.ParseUint(r.Flags["page-size"], 10, 32)
	name := strings.Join(r.CommandPath[1:], " ")
	serverDefault := name == "list" || name == "accounts list"
	if err != nil || (size == 0 && (!serverDefault || r.ChangedFlags["page-size"])) {
		return fmt.Errorf("page size must be greater than 0")
	}
	if r.Flags["cursor"] != "" {
		query.Set("cursor", r.Flags["cursor"])
	} else if size > 0 {
		query.Set("pageSize", strconv.FormatUint(size, 10))
	}
	return nil
}

func dateQuery(query url.Values, flags map[string]string, names map[string]string) error {
	for flag, param := range names {
		if value := flags[flag]; value != "" {
			date, err := time.Parse(time.RFC3339Nano, value)
			if err != nil {
				return fmt.Errorf("parsing --%s: %w", flag, err)
			}
			query.Set(param, date.Format(time.RFC3339Nano))
		}
	}
	return nil
}

func validateSegment(value, label string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." {
		return fmt.Errorf("%s must be nonempty and must not be a dot segment", label)
	}
	return nil
}

func nonnegativeInteger(value, label string) (*big.Int, error) {
	id, ok := new(big.Int).SetString(value, 10)
	if !ok || id.Sign() < 0 {
		return nil, fmt.Errorf("%s must be a nonnegative integer", label)
	}
	return id, nil
}
