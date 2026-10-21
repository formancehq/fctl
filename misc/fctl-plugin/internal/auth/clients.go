package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func executeClients(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	action := req.CommandPath[2]
	if action == "secrets" {
		return executeSecrets(ctx, client, req)
	}
	path := httpclient.Path("clients")
	if action != "create" && action != "list" {
		if err := validID(req.Args[0]); err != nil {
			return nil, err
		}
		path = httpclient.Path("clients", req.Args[0])
	}
	switch action {
	case "list", "show":
		return client.Do(ctx, http.MethodGet, path, nil, nil, nil)
	case "delete":
		return client.Do(ctx, http.MethodDelete, path, nil, nil, nil)
	case "create":
		body, err := clientFields(req, true)
		if err != nil {
			return nil, err
		}
		return client.Do(ctx, http.MethodPost, path, nil, body, nil)
	case "update":
		return updateClient(ctx, client, path, req)
	default:
		return nil, fmt.Errorf("unsupported Auth command")
	}
}

func updateClient(ctx context.Context, client *httpclient.Client, path string, req pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	patch, err := clientFields(req, false)
	if err != nil {
		return nil, err
	}
	data, err := client.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	var current struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &current); err != nil || current.Data == nil {
		return nil, fmt.Errorf("read client response must contain a data object")
	}
	// PUT replaces the options. Retain writable fields only, excluding secrets
	// and the server's ID; never replace the name with the positional client ID.
	fields := make(map[string]json.RawMessage)
	for _, field := range clientFieldFlags {
		if value, exists := current.Data[field]; exists && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			fields[field] = value
		}
	}
	if value, exists := current.Data["metadata"]; exists && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		fields["metadata"] = value
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(patch, &changes); err != nil {
		return nil, err
	}
	maps.Copy(fields, changes)
	if err := validateFields(fields, true, false); err != nil {
		return nil, fmt.Errorf("invalid client options after update: %w", err)
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, http.MethodPut, path, nil, body, nil)
}
