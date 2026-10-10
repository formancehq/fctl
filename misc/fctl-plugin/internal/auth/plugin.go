// Package auth implements the historical Auth commands through the public SDK.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

type plugin struct{ client *http.Client }

// New uses the host's HTTP client and service endpoint, including its Auth prefix.
// The manifest version identifies the legacy bundle, not the remote Auth service.
func New(client *http.Client) pluginsdk.Plugin { return &plugin{client: client} }

func (*plugin) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return pluginsdk.Manifest{}, err
	}
	return manifest(), nil
}

func (p *plugin) Execute(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	if err := ctx.Err(); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	// Remember explicit values before normalization adds defaults. The host's
	// ChangedFlags distinguishes defaults from explicit false and empty values.
	explicit := maps.Clone(req.Flags)
	changed := maps.Clone(req.ChangedFlags)
	req, err := pluginsdk.NormalizeRequest(manifest(), req)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	req.ChangedFlags = explicitFlags(explicit, changed)
	action := req.CommandPath[len(req.CommandPath)-1]
	if req.Body != nil && action != "create" && action != "update" {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("this Auth command does not accept a JSON body")
	}
	if p.client == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("auth plugin requires an HTTP client")
	}
	client, err := httpclient.New(req.Endpoint, p.client)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	var data json.RawMessage
	if req.CommandPath[1] == "users" || (len(req.CommandPath) > 2 && req.CommandPath[2] == "users") {
		data, err = executeUsers(ctx, client, req)
	} else {
		data, err = executeClients(ctx, client, req)
	}
	return pluginsdk.ExecuteResponse{Data: data}, err
}

func explicitFlags(flags map[string]string, changed map[string]bool) map[string]bool {
	result := maps.Clone(changed)
	if result == nil {
		result = make(map[string]bool)
	}
	for name, value := range flags {
		isBool := name == "public" || name == "trusted" || name == "confirm"
		if changed == nil || (value != "" && (!isBool || value != "false")) {
			result[name] = true
		}
	}
	return result
}

func validID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." {
		return fmt.Errorf("identifier must be non-empty and cannot be a dot or double dot")
	}
	return nil
}
