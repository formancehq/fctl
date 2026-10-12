// Package ledger ports cmd/ledger from fctl's historical main revision
// e00243b3e2e56aae6a09d7010b0c17890134388c onto the public plugin SDK.
package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

type plugin struct{ http *http.Client }

// New injects the host's HTTP transport. Manifest discovery works with nil.
// Endpoint is the service base URL, e.g. https://stack.example/api/ledger.
func New(client *http.Client) pluginsdk.Plugin { return &plugin{http: client} }

func (*plugin) GetManifest(context.Context) (pluginsdk.Manifest, error) { return manifest(), nil }

var _ pluginsdk.StreamingPlugin = (*plugin)(nil)

func (p *plugin) ExecuteStream(ctx context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
	request, err := pluginsdk.NormalizeRequest(manifest(), request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	name := strings.Join(request.CommandPath[1:], " ")
	if name != "import" && name != "export" {
		if streams.Input != nil || streams.Output != nil {
			return pluginsdk.ExecuteResponse{}, fmt.Errorf("%s does not accept streams", name)
		}
		return p.Execute(ctx, request)
	}
	if request.Body != nil {
		// Preserve bounded --data/direct SDK imports while files and stdin use
		// the streaming path. Inline JSON is never combined with a raw source.
		if name == "import" && streams.Input == nil && streams.Output == nil {
			return p.Execute(ctx, request)
		}
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("%s streaming does not accept Body; use ExecutionStreams", name)
	}
	if p.http == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("ledger execution requires an injected HTTP client")
	}
	client, err := httpclient.New(request.Endpoint, p.http)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if name == "import" {
		return importLogsStream(ctx, client, request, streams)
	}
	if err := validateSegment(request.Flags["ledger"], "ledger"); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	return exportLogsStream(ctx, client, request.Flags["ledger"], streams)
}

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
