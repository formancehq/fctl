// Package auth implements the Auth plugin using the public generated Auth SDK.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	sdk "github.com/formancehq/auth/pkg/client"
	"github.com/formancehq/auth/pkg/client/models/components"
	"github.com/formancehq/auth/pkg/client/models/operations"
	"github.com/formancehq/fctl/pkg/pluginsdk"
	httpadapter "github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

type plugin struct{ client *http.Client }
type response interface {
	GetHTTPMeta() components.HTTPMetadata
}

// New creates an Auth plugin with the caller's authenticated HTTP client.
func New(httpClient *http.Client) pluginsdk.Plugin { return &plugin{client: httpClient} }

func (*plugin) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return pluginsdk.Manifest{}, err
	}
	return manifest(), nil
}

func (p *plugin) Execute(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	req, err := pluginsdk.NormalizeRequest(manifest(), req)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if err = validateIDs(req.Args); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if p.client == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("auth plugin requires an HTTP client")
	}
	conn, err := httpadapter.New(req.Endpoint, p.client)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	s := sdk.New(sdk.WithServerURL(conn.Endpoint()), sdk.WithClient(conn.HTTPClient()))
	res, err := dispatch(ctx, s.Auth.V1, req)
	data, err := result(res, err)
	if err == nil && strings.Join(req.CommandPath, " ") == "auth clients secrets list" {
		data, err = secretList(data)
	}
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	return pluginsdk.ExecuteResponse{Data: data}, nil
}
func validateIDs(args []string) error {
	for _, id := range args {
		if strings.TrimSpace(id) == "" || id == "." || id == ".." {
			return fmt.Errorf("identifier must be non-empty and cannot be a dot or double dot")
		}
	}
	return nil
}
func dispatch(ctx context.Context, s *sdk.V1, req pluginsdk.ExecuteRequest) (response, error) {
	switch strings.Join(req.CommandPath, " ") {
	case "auth info":
		return s.GetServerInfo(ctx)
	case "auth discovery":
		return s.GetOIDCWellKnowns(ctx)
	case "auth clients list":
		return s.ListClients(ctx)
	case "auth clients show", "auth clients secrets list":
		return s.ReadClient(ctx, operations.ReadClientRequest{ClientID: url.PathEscape(req.Args[0])})
	case "auth clients create":
		return createClient(ctx, s, req.Body)
	case "auth clients update":
		return updateClient(ctx, s, req.Args[0], req.Body)
	case "auth clients delete":
		return s.DeleteClient(ctx, operations.DeleteClientRequest{ClientID: url.PathEscape(req.Args[0])})
	case "auth clients secrets create":
		return createSecret(ctx, s, req.Args[0], req.Body)
	case "auth clients secrets delete":
		return s.DeleteSecret(ctx, operations.DeleteSecretRequest{ClientID: url.PathEscape(req.Args[0]), SecretID: url.PathEscape(req.Args[1])})
	case "auth users list":
		return s.ListUsers(ctx)
	case "auth users show":
		return s.ReadUser(ctx, operations.ReadUserRequest{UserID: url.PathEscape(req.Args[0])})
	default:
		return nil, fmt.Errorf("unsupported auth command %q", req.CommandPath)
	}
}
func createClient(ctx context.Context, s *sdk.V1, data json.RawMessage) (response, error) {
	var req components.CreateClientRequest
	if err := decodeObject(data, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("--data requires a non-empty name")
	}
	return s.CreateClient(ctx, &req)
}
func createSecret(ctx context.Context, s *sdk.V1, id string, data json.RawMessage) (response, error) {
	var req components.CreateSecretRequest
	if err := decodeObject(data, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("--data requires a non-empty name")
	}
	return s.CreateSecret(ctx, operations.CreateSecretRequest{ClientID: url.PathEscape(id), CreateSecretRequest: &req})
}
func updateClient(ctx context.Context, s *sdk.V1, id string, data json.RawMessage) (response, error) {
	var patch components.UpdateClientRequest
	if err := decodeObject(data, &patch); err != nil {
		return nil, err
	}
	current, err := s.ReadClient(ctx, operations.ReadClientRequest{ClientID: url.PathEscape(id)})
	if err != nil {
		return current, err
	}
	body, err := responseBody(current)
	if err != nil {
		return nil, err
	}
	merged, err := mergeOptions(body, data)
	if err != nil {
		return nil, err
	}
	var req components.UpdateClientRequest
	if err = decodeObject(merged, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("--data requires a non-empty name")
	}
	return s.UpdateClient(ctx, operations.UpdateClientRequest{ClientID: url.PathEscape(id), UpdateClientRequest: &req})
}

// Auth errors leave stdout empty; the host reports the error separately.
func result(res response, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, err
	}
	return responseBody(res)
}
