package ledger

import (
	"context"
	"encoding/json"
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

func prepareTransactions(ctx context.Context, client *httpclient.Client, name string, r pluginsdk.ExecuteRequest) (operation, error) {
	op := operation{method: http.MethodGet, query: make(url.Values)}
	switch name {
	case "send":
		body, err := sendPayload(r)
		op.method, op.path, op.body = http.MethodPost, httpclient.Path(r.Flags["ledger"], "transactions"), body
		return op, err
	case "transactions list":
		return transactionList(r)
	case "transactions num":
		body, err := numPayload(r)
		op.method, op.path, op.body = http.MethodPost, httpclient.Path(r.Flags["ledger"], "transactions"), body
		return op, err
	case "transactions show", "transactions revert":
		return transactionOperation(ctx, client, name, r)
	case "transactions set-metadata", "transactions delete-metadata":
		return metadataOperation(ctx, client, "transactions", r)
	default:
		return op, fmt.Errorf("unknown ledger command %q", name)
	}
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

func nonnegativeInteger(value, label string) (*big.Int, error) {
	id, ok := new(big.Int).SetString(value, 10)
	if !ok || id.Sign() < 0 {
		return nil, fmt.Errorf("%s must be a nonnegative integer", label)
	}
	return id, nil
}

func sendPayload(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	source := "world"
	args := r.Args
	if len(args) == 4 {
		source, args = args[0], args[1:]
	}
	if strings.TrimSpace(source) == "" || strings.TrimSpace(args[0]) == "" || strings.TrimSpace(args[2]) == "" {
		return nil, fmt.Errorf("source, destination and asset must be nonempty")
	}
	amount, err := nonnegativeInteger(args[1], "amount")
	if err != nil {
		return nil, err
	}
	metadata, err := flagPairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"postings":  []map[string]any{{"source": source, "destination": args[0], "amount": amount, "asset": args[2]}},
		"reference": r.Flags["reference"],
	}
	if len(metadata) > 0 {
		body["metadata"] = metadata
	}
	return json.Marshal(body)
}

func numPayload(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	body, err := scriptBody(r.Body)
	if err != nil {
		return nil, err
	}
	script, err := scriptWithVariables(body["script"], r.Flags)
	if err != nil {
		return nil, err
	}
	body["script"] = script
	if err := transactionBodyFlags(body, r); err != nil {
		return nil, err
	}
	return json.Marshal(body)
}

func scriptWithVariables(raw json.RawMessage, flags map[string]string) (json.RawMessage, error) {
	var script map[string]json.RawMessage
	if err := json.Unmarshal(raw, &script); err != nil || script == nil {
		return nil, fmt.Errorf("numscript body must contain a script object")
	}
	var plain string
	if err := json.Unmarshal(script["plain"], &plain); err != nil || strings.TrimSpace(plain) == "" {
		return nil, fmt.Errorf("numscript plain text must be nonempty; the host must read the source into Body")
	}
	vars := map[string]json.RawMessage{}
	if raw, exists := script["vars"]; exists {
		if err := json.Unmarshal(raw, &vars); err != nil || vars == nil {
			return nil, fmt.Errorf("script vars must be a JSON object")
		}
	}
	if err := applyVariables(vars, flags); err != nil {
		return nil, err
	}
	if len(vars) > 0 {
		data, err := json.Marshal(vars)
		if err != nil {
			return nil, err
		}
		script["vars"] = data
	}
	return json.Marshal(script)
}

func applyVariables(vars map[string]json.RawMessage, flags map[string]string) error {
	for _, flag := range []string{"account-var", "portion-var", "amount-var"} {
		entries, err := flagPairs(flags[flag])
		if err != nil {
			return fmt.Errorf("--%s: %w", flag, err)
		}
		for name, value := range entries {
			data, err := variableJSON(flag, value)
			if err != nil {
				return fmt.Errorf("--%s variable %s: %w", flag, name, err)
			}
			vars[name] = data
		}
	}
	return nil
}

func variableJSON(flag, value string) (json.RawMessage, error) {
	if flag != "amount-var" {
		return json.Marshal(value)
	}
	amountValue, asset, ok := strings.Cut(value, "/")
	if !ok || asset == "" {
		return nil, fmt.Errorf("expected name=amount/asset")
	}
	amount, err := nonnegativeInteger(amountValue, "amount variable")
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"amount": amount, "asset": asset})
}

func scriptBody(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("numscript Body is required; file/stdin reads belong to the host")
	}
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		script, err := json.Marshal(map[string]string{"plain": plain})
		return map[string]json.RawMessage{"script": script}, err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		return nil, fmt.Errorf("numscript Body must be a JSON string or object")
	}
	if _, hasPlain := body["plain"]; hasPlain {
		return map[string]json.RawMessage{"script": raw}, nil
	}
	if _, exists := body["postings"]; exists {
		return nil, fmt.Errorf("numscript Body cannot also contain postings")
	}
	return body, nil
}

func transactionBodyFlags(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	if err := metadataBodyFlag(body, r); err != nil {
		return err
	}
	if err := referenceBodyFlag(body, r); err != nil {
		return err
	}
	return timestampBodyFlag(body, r)
}

func metadataBodyFlag(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	metadata, err := flagPairs(r.Flags["metadata"])
	if err != nil {
		return err
	}
	if len(metadata) > 0 || r.ChangedFlags["metadata"] {
		body["metadata"], err = json.Marshal(metadata)
		if err != nil {
			return err
		}
	} else if raw, exists := body["metadata"]; exists {
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil || values == nil {
			return fmt.Errorf("transaction metadata must be a JSON object")
		}
	}
	return nil
}

func referenceBodyFlag(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	if _, exists := body["reference"]; !exists || r.Flags["reference"] != "" || r.ChangedFlags["reference"] {
		data, err := json.Marshal(r.Flags["reference"])
		if err != nil {
			return err
		}
		body["reference"] = data
	}
	var reference string
	if err := json.Unmarshal(body["reference"], &reference); err != nil {
		return fmt.Errorf("transaction reference must be a string")
	}
	return nil
}

func timestampBodyFlag(body map[string]json.RawMessage, r pluginsdk.ExecuteRequest) error {
	if r.Flags["timestamp"] != "" {
		data, err := json.Marshal(r.Flags["timestamp"])
		if err != nil {
			return err
		}
		body["timestamp"] = data
	} else if r.ChangedFlags["timestamp"] {
		delete(body, "timestamp")
	}
	if raw, exists := body["timestamp"]; exists {
		var timestamp string
		if err := json.Unmarshal(raw, &timestamp); err != nil {
			return fmt.Errorf("transaction timestamp must be a string")
		}
		if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
			return fmt.Errorf("parsing transaction timestamp: %w", err)
		}
	}
	return nil
}
