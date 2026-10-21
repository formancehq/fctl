package ledger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

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

func validateSchema(raw json.RawMessage) error {
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(raw, &schema); err != nil || schema == nil {
		return fmt.Errorf("schema Body must be a JSON object; the host must read and convert the source")
	}
	for _, name := range []string{"chart", "queries", "transactions"} {
		value, exists := schema[name]
		if !exists {
			if name == "chart" {
				return fmt.Errorf("schema chart is required")
			}
			continue
		}
		if err := validateSchemaEntries(name, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSchemaEntries(name string, raw json.RawMessage) error {
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil || entries == nil {
		return fmt.Errorf("schema %s must be a JSON object", name)
	}
	for key, entry := range entries {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(entry, &object); err != nil || object == nil {
			return fmt.Errorf("schema %s[%s] must be a JSON object", name, key)
		}
	}
	return nil
}
