package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
	"github.com/formancehq/fctl/pkg/pluginsdk/transport"
)

// ExternalFactory uses cached metadata for help and forms. Processes exist only
// during Execute, including interactive choice queries, and are always closed.
// Authentication is injected through the same host HTTP boundary as embedded
// plugins; credentials are never passed in the executable's arguments.
func ExternalFactory(binary string, manifest pluginsdk.Manifest) Factory {
	return ExternalFactoryWithVerifier(manifest, func(context.Context) (string, error) { return binary, nil })
}

// ExternalFactoryWithVerifier keeps metadata independent of executable
// availability. The verifier runs before every execution, including choice
// requests, and may verify integrity or locate a cached binary.
func ExternalFactoryWithVerifier(manifest pluginsdk.Manifest, verify func(context.Context) (string, error)) Factory {
	return func(client *http.Client) pluginsdk.Plugin {
		return &external{verify: verify, manifest: cloneManifest(manifest), http: client}
	}
}

type external struct {
	verify   func(context.Context) (string, error)
	manifest pluginsdk.Manifest
	http     *http.Client
	mu       sync.Mutex
	checked  string
}

func (p *external) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return pluginsdk.Manifest{}, err
	}
	return cloneManifest(p.manifest), nil
}

func (p *external) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (response pluginsdk.ExecuteResponse, err error) {
	binary, err := p.verify(ctx)
	if err != nil {
		return response, fmt.Errorf("verify plugin executable: %w", err)
	}
	if err := p.checkVersion(ctx, request.Endpoint); err != nil {
		return response, err
	}
	client, err := transport.Open(ctx, binary, p.http, request.Endpoint)
	if err != nil {
		return response, err
	}
	defer func() { err = errors.Join(err, client.Close()) }()
	manifest, err := client.GetManifest(ctx)
	if err != nil {
		return response, err
	}
	if !reflect.DeepEqual(manifest, p.manifest) {
		return response, fmt.Errorf("plugin manifest differs from its installed catalogue; run fctl plugins sync")
	}
	return client.Execute(ctx, request)
}

func (p *external) checkVersion(ctx context.Context, endpoint string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.checked != "" && p.checked == endpoint {
		return nil
	}
	client, err := httpclient.New(endpoint, p.http)
	if err != nil {
		return err
	}
	data, err := client.Do(ctx, http.MethodGet, "/_info", nil, nil, nil)
	if err != nil {
		return fmt.Errorf("check %s plugin version: %w", p.manifest.Service, err)
	}
	version, err := pluginsdk.ServiceVersion(data)
	if err != nil {
		return fmt.Errorf("decode %s version: %w", p.manifest.Service, err)
	}
	if version != p.manifest.Version {
		return fmt.Errorf("installed plugin targets %s %s, service reports %s; run fctl plugins sync or install the matching binary", p.manifest.Service, p.manifest.Version, version)
	}
	p.checked = endpoint
	return nil
}
