package pluginhost

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

// PrepareService selects an external provider without starting it for cached help.
func PrepareService(ctx context.Context, root *cobra.Command, settings *connection.Settings, args []string, service string) (plugin.Factory, error) {
	descriptor, err := supportedPluginService(service)
	if err != nil {
		return nil, err
	}
	plan, err := parsePluginBootstrap(root, args)
	if err != nil {
		return nil, err
	}
	manager, err := pluginManager(settings, root)
	if err != nil {
		return nil, err
	}
	prep := servicePreparation{root: root, settings: settings, manager: manager, service: descriptor}
	lock, err := prep.resolve(ctx, plan)
	if errors.Is(err, pluginmanager.ErrNotInstalled) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("prepare %s plugin: %w", descriptor.title, err)
	}
	if err := validateServiceManifest(ctx, root, service, lock.Manifest); err != nil {
		return nil, err
	}
	return plugin.ExternalFactoryWithVerifier(lock.Manifest, func(ctx context.Context) (string, error) {
		return manager.BinaryContext(ctx, lock)
	}), nil
}

type pluginBootstrap struct {
	commands []string
	help     bool
}

func parsePluginBootstrap(root *cobra.Command, args []string) (pluginBootstrap, error) {
	bootstrap := pflag.NewFlagSet("plugin selection", pflag.ContinueOnError)
	bootstrap.SetOutput(io.Discard)
	bootstrap.AddFlagSet(root.PersistentFlags())
	bootstrap.ParseErrorsWhitelist.UnknownFlags = true
	var plan pluginBootstrap
	bootstrap.BoolVarP(&plan.help, "help", "h", false, "")
	err := bootstrap.Parse(args)
	plan.commands = bootstrap.Args()
	return plan, err
}

type servicePreparation struct {
	service  serviceDescriptor
	root     *cobra.Command
	settings *connection.Settings
	manager  *pluginmanager.Manager
}

func (p servicePreparation) resolve(ctx context.Context, plan pluginBootstrap) (pluginmanager.Lock, error) {
	lock, loadErr := p.cached()
	catalogue := os.Getenv("FCTL_PLUGIN_CATALOGUE")
	if catalogue == "" && loadErr == nil {
		catalogue = lock.Catalogue
	}
	if len(plan.commands) == 1 && plan.commands[0] == p.service.name {
		return lock, loadErr
	}
	if catalogue != "" && !plan.help && len(plan.commands) > 0 && plan.commands[0] == p.service.name {
		return p.matchVersion(ctx, catalogue, nil)
	}
	if catalogue == "" && errors.Is(loadErr, pluginmanager.ErrNotInstalled) && !plan.help &&
		len(plan.commands) > 0 && plan.commands[0] == p.service.name {
		return p.discover(ctx)
	}
	return lock, loadErr
}

func (p servicePreparation) discover(ctx context.Context) (pluginmanager.Lock, error) {
	catalogue, err := p.manager.Discover(ctx, pluginmanager.DefaultCatalogue)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	for _, release := range catalogue.Releases {
		if release.Service == p.service.name && release.Platform == pluginmanager.CurrentPlatform() {
			return p.matchVersion(ctx, pluginmanager.DefaultCatalogue, &catalogue)
		}
	}
	return pluginmanager.Lock{}, fmt.Errorf("%w: %s has no published plugin for %s; prepare it with fctl plugins sync --service %s or plugins install --service %s --binary PATH", pluginmanager.ErrNoRelease, p.service.title, pluginmanager.CurrentPlatform(), p.service.name, p.service.name)
}

func (p servicePreparation) cached() (pluginmanager.Lock, error) {
	return cachedServiceLock(p.settings, p.root, p.manager, p.service.name)
}

func (p servicePreparation) matchVersion(ctx context.Context, catalogue string, discovered *pluginmanager.Catalogue) (pluginmanager.Lock, error) {
	target, version, err := pluginServiceSyncTarget(ctx, p.settings, p.root, p.service.name, "")
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	lock, err := p.manager.Load(target, p.service.name)
	if err == nil && lock.ServiceVersion == version {
		return lock, nil
	}
	if err != nil && !errors.Is(err, pluginmanager.ErrNotInstalled) {
		return lock, err
	}
	var release pluginmanager.Release
	if discovered != nil {
		release, err = discovered.Resolve(pluginmanager.CurrentPlatform(), p.service.name, version, 0)
	} else {
		release, err = p.manager.Resolve(ctx, catalogue, p.service.name, version, 0)
	}
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	if err := validateServiceManifest(ctx, p.root, p.service.name, release.Manifest); err != nil {
		return pluginmanager.Lock{}, err
	}
	lock, err = p.manager.InstallResolved(ctx, catalogue, target, release)
	if err != nil {
		return lock, err
	}
	_, err = fmt.Fprintf(p.root.ErrOrStderr(), "Prepared %s plugin %s (revision %d).\n", p.service.title, lock.ServiceVersion, lock.Revision)
	return lock, err
}
