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
	manifest, err := p.GetManifest(ctx)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	request.CommandPath = normalized.CommandPath
	if p.http == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("legacy plugin requires an injected HTTP client")
	}
	client, err := httpclient.New(request.Endpoint, p.http)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	data, err := client.Do(ctx, http.MethodGet, "/_info", nil, nil, nil)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("verify legacy %s version: %w", p.service, err)
	}
	version, err := pluginsdk.ServiceVersion(data)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if !Supported(p.service, version, request.Context["stack-version"]) {
		return pluginsdk.ExecuteResponse{}, Unsupported(p.service, version, request.Context["stack-version"])
	}
	return p.plugin.Execute(ctx, request)
}
