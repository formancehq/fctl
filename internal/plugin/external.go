package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
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
	return func(client *http.Client) pluginsdk.Plugin {
		return &external{binary: binary, manifest: cloneManifest(manifest), http: client}
	}
}

type external struct {
	binary   string
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
	if err := p.checkVersion(ctx, request.Endpoint); err != nil {
		return response, err
	}
	client, err := transport.Open(ctx, p.binary, p.http, request.Endpoint)
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
	if p.checked == endpoint {
		return nil
	}
	client, err := httpclient.New(endpoint, p.http)
	if err != nil {
		return err
	}
	data, err := client.Do(ctx, http.MethodGet, "/_info", nil, nil, nil)
	if err != nil {
		return fmt.Errorf("check Ledger plugin version: %w", err)
	}
	var info struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return fmt.Errorf("decode Ledger version: %w", err)
	}
	version := strings.TrimPrefix(info.Version, "v")
	if version != p.manifest.Version {
		return fmt.Errorf("installed plugin targets Ledger %s, service reports %s; run fctl plugins sync or install the matching binary", p.manifest.Version, version)
	}
	p.checked = endpoint
	return nil
}
