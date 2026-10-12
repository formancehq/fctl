package legacy

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

// NewService exposes one service facade with the same SDK contract as New.
func NewService(service string, client *http.Client) pluginsdk.Plugin {
	return newService(service, client)
}

func newService(service string, client *http.Client) *servicePlugin {
	factory, ok := Factories()[service]
	if !ok {
		return &servicePlugin{service: service, http: client}
	}
	return &servicePlugin{service: service, http: client, plugin: factory(client)}
}

type servicePlugin struct {
	service string
	http    *http.Client
	plugin  pluginsdk.Plugin
}

func (p *servicePlugin) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	if p.plugin == nil {
		return pluginsdk.Manifest{}, fmt.Errorf("unknown legacy service %q", p.service)
	}
	manifest, err := p.plugin.GetManifest(ctx)
	if err == nil {
		manifest.Version = Version
		manifest.Root.Short += " (legacy API)"
	}
	return manifest, err
}

func (p *servicePlugin) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	request, err := p.prepareRequest(ctx, request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	return p.plugin.Execute(ctx, request)
}

var _ pluginsdk.StreamingPlugin = (*servicePlugin)(nil)

func (p *servicePlugin) ExecuteStream(ctx context.Context, request pluginsdk.ExecuteRequest, streams pluginsdk.ExecutionStreams) (pluginsdk.ExecuteResponse, error) {
	request, err := p.prepareRequest(ctx, request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if streaming, ok := p.plugin.(pluginsdk.StreamingPlugin); ok {
		return streaming.ExecuteStream(ctx, request, streams)
	}
	if streams.Input != nil || streams.Output != nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("legacy %s does not accept streams", p.service)
	}
	return p.plugin.Execute(ctx, request)
}

// Both entry points verify compatibility before consuming streams or mutating.
func (p *servicePlugin) prepareRequest(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteRequest, error) {
	manifest, err := p.GetManifest(ctx)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	request.CommandPath = normalized.CommandPath
	if p.http == nil {
		return pluginsdk.ExecuteRequest{}, fmt.Errorf("legacy plugin requires an injected HTTP client")
	}
	client, err := httpclient.New(request.Endpoint, p.http)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	data, err := client.Do(ctx, http.MethodGet, "/_info", nil, nil, nil)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, fmt.Errorf("verify legacy %s version: %w", p.service, err)
	}
	version, err := pluginsdk.ServiceVersion(data)
	if err != nil {
		return pluginsdk.ExecuteRequest{}, err
	}
	if !Supported(p.service, version, request.Context["stack-version"]) {
		return pluginsdk.ExecuteRequest{}, Unsupported(p.service, version, request.Context["stack-version"])
	}
	return request, nil
}
