package legacy

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/auth"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/ledger"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/orchestration"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/payments"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/reconciliation"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/wallets"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/webhooks"
)

type Factory func(*http.Client) pluginsdk.Plugin

func Factories() map[string]Factory {
	return map[string]Factory{
		"auth":           auth.New,
		"ledger":         ledger.New,
		"orchestration":  orchestration.New,
		"payments":       payments.New,
		"reconciliation": reconciliation.New,
		"wallets":        wallets.New,
		"webhooks":       webhooks.New,
	}
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
