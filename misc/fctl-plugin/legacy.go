package legacy

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	"github.com/formancehq/fctl/misc/fctl-plugin/auth"
	"github.com/formancehq/fctl/misc/fctl-plugin/ledger"
	"github.com/formancehq/fctl/misc/fctl-plugin/modules"
)

type Factory func(*http.Client) pluginsdk.Plugin

func Factories() map[string]Factory {
	out := map[string]Factory{"ledger": ledger.New, "auth": auth.New}
	for service, factory := range modules.Factories() {
		out[service] = factory
	}
	return out
}

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

type bundle struct{ http *http.Client }

// New exposes all legacy services in one independently distributable plugin.
func New(client *http.Client) pluginsdk.Plugin { return &bundle{http: client} }

func (p *bundle) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	manifest := pluginsdk.Manifest{Name: "legacy", Service: "stack", Version: Version, ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{Use: "legacy", Short: "Historical service commands for stacks before v4", Target: "stack"}}
	factories := Factories()
	for _, service := range slices.Sorted(maps.Keys(factories)) {
		metadata, err := NewService(service, nil).GetManifest(ctx)
		if err != nil {
			return pluginsdk.Manifest{}, err
		}
		metadata.Root.Service = service
		prefixSources(&metadata.Root)
		manifest.Root.Subcommands = append(manifest.Root.Subcommands, metadata.Root)
	}
	return manifest, nil
}

func prefixSources(command *pluginsdk.CommandSpec) {
	for i := range command.Inputs {
		if source := command.Inputs[i].Source; source != nil {
			source.CommandPath = append([]string{"legacy"}, source.CommandPath...)
		}
	}
	for i := range command.Subcommands {
		prefixSources(&command.Subcommands[i])
	}
}

func (p *bundle) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	manifest, err := p.GetManifest(ctx)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	normalized, err := pluginsdk.NormalizeRequest(manifest, request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	request.CommandPath = normalized.CommandPath
	if len(request.CommandPath) < 2 {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("select a legacy service")
	}
	request.CommandPath = slices.Clone(request.CommandPath[1:])
	return NewService(request.CommandPath[0], p.http).Execute(ctx, request)
}
