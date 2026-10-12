package factory

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	legacy "github.com/formancehq/fctl/misc/fctl-plugin"
	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/internal/pluginhost"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
	"github.com/formancehq/fctl/v4/internal/pluginselection"
)

func legacyFactory(service string) plugin.Factory {
	return func(client *http.Client) pluginsdk.Plugin { return legacy.NewService(service, client) }
}

type providerPreparation struct {
	root      *cobra.Command
	settings  *connection.Settings
	service   string
	directory string
	modern    bool
	execution bool
}

func prepareProvider(ctx context.Context, root *cobra.Command, settings *connection.Settings, args []string, service string) (plugin.Factory, bool, error) {
	selected, metadataOnly, err := pluginhost.SelectedCommand(root, args)
	if err != nil {
		return nil, false, err
	}
	modern := service == "auth" || service == "ledger"
	if modern && os.Getenv("FCTL_PLUGIN_CATALOGUE") != "" {
		if err := command.ValidateOutputOptions(settings.Output, settings.Color); err != nil {
			return nil, false, err
		}
		provider, err := pluginhost.PrepareService(ctx, root, settings, args, service)
		return provider, false, err
	}
	dir, err := settings.DirectoryPath()
	if err != nil {
		return nil, false, err
	}
	p := providerPreparation{root: root, settings: settings, service: service, directory: filepath.Join(dir, "plugins"), modern: modern, execution: legacyServiceCommand(ctx, selected, service) && !metadataOnly}
	return p.prepare(ctx, args)
}

func (p providerPreparation) selection(ctx context.Context) (pluginselection.Selection, error) {
	if p.execution {
		return p.inspect(ctx)
	}
	target, err := pluginhost.SelectionTarget(p.settings, p.root, p.service)
	if err != nil {
		return pluginselection.Selection{}, err
	}
	return pluginselection.Load(p.directory, target, p.service)
}

func (p providerPreparation) inspect(ctx context.Context) (pluginselection.Selection, error) {
	snapshot, err := pluginhost.InspectService(ctx, p.root, p.settings, p.service)
	if err != nil {
		return snapshot, err
	}
	historical, err := p.useLegacy(snapshot)
	if err != nil {
		return snapshot, err
	}
	if historical {
		snapshot.Provider = "legacy"
		snapshot.PluginVersion = legacy.Version
	} else {
		snapshot.Provider = "modern"
	}
	err = pluginselection.Save(p.directory, snapshot)
	return snapshot, err
}

func (p providerPreparation) useLegacy(snapshot pluginselection.Selection) (bool, error) {
	if !legacy.Supported(p.service, snapshot.ServiceVersion, snapshot.StackVersion) {
		return false, nil
	}
	if !p.modern || snapshot.StackVersion != "" {
		return true, nil
	}
	_, err := pluginhost.Show(p.settings, p.root, p.service)
	if errors.Is(err, pluginmanager.ErrNotInstalled) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

// LegacyBundle exposes explicit legacy commands through the same SDK. The
// request resolver checks the Cloud compatibility line before every operation.
func LegacyBundle() plugin.Factory { return legacy.New }

func (p providerPreparation) checkModernMetadata(selected pluginselection.Selection) error {
	lock, err := pluginhost.Show(p.settings, p.root, p.service)
	if errors.Is(err, pluginmanager.ErrNotInstalled) {
		return nil
	}
	if err != nil {
		return err
	}
	if lock.ServiceVersion != selected.ServiceVersion {
		return fmt.Errorf("%s %s was observed but its plugin is not prepared; run fctl plugins sync --service %s", p.service, selected.ServiceVersion, p.service)
	}
	return nil
}

func legacyServiceCommand(ctx context.Context, selected, service string) bool {
	if selected == service {
		return true
	}
	manifest, err := legacy.NewService(service, nil).GetManifest(ctx)
	return err == nil && slices.Contains(manifest.Root.Aliases, selected)
}

func (p providerPreparation) modernProvider(ctx context.Context, args []string, selected pluginselection.Selection, observed bool) (plugin.Factory, bool, error) {
	if observed && !p.execution {
		if err := p.checkModernMetadata(selected); err != nil {
			return nil, false, err
		}
	}
	provider, err := pluginhost.PrepareService(ctx, p.root, p.settings, args, p.service)
	return provider, false, err
}

func (p providerPreparation) prepare(ctx context.Context, args []string) (plugin.Factory, bool, error) {
	if p.execution {
		if err := command.ValidateOutputOptions(p.settings.Output, p.settings.Color); err != nil {
			return nil, false, err
		}
		provider, handled, err := p.installedModern(ctx, args)
		if handled || err != nil {
			return provider, false, err
		}
	}
	return p.selectProvider(ctx, args)
}
func (p providerPreparation) selectProvider(ctx context.Context, args []string) (plugin.Factory, bool, error) {
	selectedProvider, err := p.selection(ctx)
	if err != nil && !errors.Is(err, pluginselection.ErrNotSelected) {
		return nil, false, err
	}
	if err == nil && selectedProvider.Provider == "legacy" {
		if !legacy.Supported(p.service, selectedProvider.ServiceVersion, selectedProvider.StackVersion) {
			return nil, false, fmt.Errorf("cached legacy selection is outside the declared compatibility table")
		}
		return legacyFactory(p.service), true, nil
	}
	if p.modern {
		return p.modernProvider(ctx, args, selectedProvider, err == nil)
	}
	if p.execution {
		return nil, false, fmt.Errorf("%s has no modern plugin; legacy commands require a supported service on a stack before v4", p.service)
	}
	return nil, false, nil
}

// A prepared modern provider stays modern after version drift. Its existing
// verifier/discovery performs the check with the execution transport, avoiding
// a second OAuth grant and preserving verification-before-network behavior.
func (p providerPreparation) installedModern(ctx context.Context, args []string) (plugin.Factory, bool, error) {
	if !p.modern {
		return nil, false, nil
	}
	options, _, _, _, err := p.settings.Resolve(p.root)
	if err != nil {
		return nil, false, err
	}
	if options.AuthMode == "cloud" {
		return nil, false, nil
	}
	_, err = pluginhost.Show(p.settings, p.root, p.service)
	if errors.Is(err, pluginmanager.ErrNotInstalled) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	provider, _, err := p.modernProvider(ctx, args, pluginselection.Selection{}, false)
	if err != nil || provider == nil {
		return provider, true, err
	}
	err = p.saveModernMetadata(ctx, provider)
	return provider, true, err
}
func (p providerPreparation) saveModernMetadata(ctx context.Context, provider plugin.Factory) error {
	manifest, err := provider(nil).GetManifest(ctx)
	if err != nil {
		return err
	}
	target, err := pluginhost.SelectionTarget(p.settings, p.root, p.service)
	if err != nil {
		return err
	}
	return pluginselection.Save(p.directory, pluginselection.Selection{Target: target, Service: p.service, ServiceVersion: manifest.Version, Provider: "modern"})
}
