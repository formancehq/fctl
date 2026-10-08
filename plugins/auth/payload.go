package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"reflect"
)

func decodeObject(data json.RawMessage, target any) error {
	if len(bytes.TrimSpace(data)) == 0 || bytes.TrimSpace(data)[0] != '{' {
		return fmt.Errorf("--data must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid --data: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("--data field %s cannot be null; use false, an empty string, array or object", key)
		}
	}
	return nil
}
func responseBody(res response) (json.RawMessage, error) {
	if res == nil || (reflect.ValueOf(res).Kind() == reflect.Pointer && reflect.ValueOf(res).IsNil()) {
		return nil, fmt.Errorf("auth SDK returned no response")
	}
	httpRes := res.GetHTTPMeta().Response
	if httpRes == nil {
		return nil, fmt.Errorf("auth SDK returned no HTTP response")
	}
	if httpRes.Body == nil {
		return json.RawMessage("null"), nil
	}
	body, err := io.ReadAll(io.LimitReader(httpRes.Body, 32<<20+1))
	closeErr := httpRes.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read Auth response: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Auth response: %w", closeErr)
	}
	if len(body) > 32<<20 {
		return nil, fmt.Errorf("auth response exceeds 32 MiB")
	}
	if httpRes.StatusCode < http.StatusOK || httpRes.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("auth HTTP %d", httpRes.StatusCode)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("auth returned invalid JSON")
	}
	return body, nil
}
func mergeOptions(current, patch json.RawMessage) (json.RawMessage, error) {
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(current, &envelope); err != nil {
		return nil, err
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("auth response is missing client data")
	}
	fields := make(map[string]json.RawMessage)
	for _, key := range []string{"name", "description", "public", "trusted", "redirectUris", "postLogoutRedirectUris", "scopes", "metadata"} {
		if value, ok := envelope.Data[key]; ok && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			fields[key] = value
		}
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(patch, &changes); err != nil {
		return nil, err
	}
	maps.Copy(fields, changes)
	return json.Marshal(fields)
}

func secretList(body json.RawMessage) (json.RawMessage, error) {
	var envelope struct {
		Data *struct {
			Secrets json.RawMessage `json:"secrets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("auth response is missing client data")
	}
	if len(envelope.Data.Secrets) == 0 || bytes.Equal(envelope.Data.Secrets, []byte("null")) {
		return json.RawMessage("[]"), nil
	}
	return envelope.Data.Secrets, nil
}
