// Package connectivity implements the Connectivity API through the public plugin SDK.
package connectivity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

type plugin struct{ client *http.Client }

// New creates a plugin with the host's authenticated client. Metadata needs no client.
func New(client *http.Client) pluginsdk.Plugin { return &plugin{client: client} }

func (*plugin) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return pluginsdk.Manifest{}, err
	}
	return manifest(), nil
}

type operation struct {
	id, command, method string
	segments            []string
	list, query         bool
}

// Contract: connectivity-api/openapi.yaml at ce2324887f5b5ec4e3c2ec934ac874656d4c5348.
var operations = []operation{
	{"getQueryCapabilities", "query capabilities", http.MethodGet, []string{"_query", "capabilities"}, false, false},
	{"healthcheck", "health", http.MethodGet, []string{"_healthcheck"}, false, false},
	{"info", "info", http.MethodGet, []string{"_info"}, false, false},
	{"listConnectors", "connectors list", http.MethodGet, []string{"connectors"}, true, true},
	{"getConnectorFacets", "connectors facets", http.MethodGet, []string{"connectors", "_facets"}, false, true},
	{"getConnector", "connectors show", http.MethodGet, []string{"connectors", "$0"}, false, false},
	{"listConnectorVersions", "connectors versions list", http.MethodGet, []string{"connectors", "$0", "versions"}, true, true},
	{"getConnectorVersion", "connectors versions show", http.MethodGet, []string{"connectors", "$0", "versions", "$1"}, false, false},
	{"listConnectorInstances", "instances list", http.MethodGet, []string{"connectorinstances"}, true, true},
	{"createConnectorInstance", "instances create", http.MethodPost, []string{"connectorinstances"}, false, false},
	{"getConnectorInstance", "instances show", http.MethodGet, []string{"connectorinstances", "$0"}, false, false},
	{"replaceConnectorInstance", "instances replace", http.MethodPut, []string{"connectorinstances", "$0"}, false, false},
	{"patchConnectorInstance", "instances patch", http.MethodPatch, []string{"connectorinstances", "$0"}, false, false},
	{"deleteConnectorInstance", "instances delete", http.MethodDelete, []string{"connectorinstances", "$0"}, false, false},
}

func (p *plugin) Execute(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	req, err := pluginsdk.NormalizeRequest(manifest(), req)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	command := strings.Join(req.CommandPath[1:], " ")
	for _, op := range operations {
		if command != op.command {
			continue
		}
		return p.execute(ctx, op, req)
	}
	return pluginsdk.ExecuteResponse{}, fmt.Errorf("unsupported connectivity command %q", command)
}

func (p *plugin) execute(ctx context.Context, op operation, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	if err := validateRequest(op, req); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if p.client == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("connectivity plugin requires an HTTP client")
	}
	client, err := httpclient.New(req.Endpoint, p.client)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	headers := http.Header{}
	if op.method == http.MethodPatch {
		headers.Set("Content-Type", "application/merge-patch+json")
	}
	data, err := client.Do(ctx, op.method, operationPath(op, req.Args), operationQuery(op, req.Flags), req.Body, headers)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, serviceError(err)
	}
	return pluginsdk.ExecuteResponse{Data: data}, nil
}

func operationPath(op operation, args []string) string {
	segments := make([]string, len(op.segments))
	for i, segment := range op.segments {
		switch segment {
		case "$0":
			segments[i] = args[0]
		case "$1":
			segments[i] = args[1]
		default:
			segments[i] = segment
		}
	}
	return httpclient.Path(segments...)
}

func operationQuery(op operation, flags map[string]string) url.Values {
	query := url.Values{}
	if op.query && flags["query"] != "" {
		query.Set("query", flags["query"])
	}
	if op.list {
		query.Set("pageSize", flags["page-size"])
		if flags["cursor"] != "" {
			query.Set("cursor", flags["cursor"])
		}
	}
	return query
}

func serviceError(err error) error {
	// Connectivity uses code/message rather than Ledger's errorCode/errorMessage.
	var failure *httpclient.Error
	if errors.As(err, &failure) {
		var details struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(failure.Body, &details) == nil && details.Code != "" {
			failure.Code, failure.Message = details.Code, details.Message
		}
	}
	return err
}
