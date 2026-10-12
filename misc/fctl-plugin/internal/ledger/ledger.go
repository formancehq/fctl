package ledger

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

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
	case "accounts list", "accounts show", "accounts set-metadata", "accounts delete-metadata":
		return prepareAccounts(ctx, client, name, r)
	case "send", "transactions list", "transactions num", "transactions show", "transactions revert", "transactions set-metadata", "transactions delete-metadata":
		return prepareTransactions(ctx, client, name, r)
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

func flagPairs(value string) (map[string]string, error) {
	if value == "" {
		return map[string]string{}, nil
	}
	var entries []string
	if strings.HasPrefix(strings.TrimSpace(value), "[") {
		if err := json.Unmarshal([]byte(value), &entries); err != nil {
			return nil, fmt.Errorf("expected a JSON string array: %w", err)
		}
	} else {
		reader := csv.NewReader(strings.NewReader(value))
		reader.FieldsPerRecord = -1
		var err error
		entries, err = reader.Read()
		if err != nil {
			return nil, fmt.Errorf("parse comma-separated pairs: %w", err)
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("use a JSON string array for multiline values")
		}
	}
	return pairs(entries)
}

func pairs(entries []string) (map[string]string, error) {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("malformed key=value pair %q", entry)
		}
		result[key] = value
	}
	return result, nil
}
