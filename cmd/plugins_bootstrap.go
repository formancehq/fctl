package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func prepareLedgerPlugin(ctx context.Context, root *cobra.Command, settings *connection.Settings, args []string) (plugin.Factory, error) {
	return prepareServicePlugin(ctx, root, settings, args, "ledger")
}

func prepareServicePlugin(ctx context.Context, root *cobra.Command, settings *connection.Settings, args []string, service string) (plugin.Factory, error) {
	descriptor, err := supportedPluginService(service)
	if err != nil {
		return nil, err
	}
	if args == nil && service == "ledger" {
		return nil, nil
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

// The zero service descriptor retains the Ledger pilot helpers' behavior.
type ledgerPreparation = servicePreparation

type servicePreparation struct {
	service  serviceDescriptor
	root     *cobra.Command
	settings *connection.Settings
	manager  *pluginmanager.Manager
}

func (p servicePreparation) descriptor() serviceDescriptor {
	if p.service.name == "" {
		return serviceDescriptor{name: "ledger", title: "Ledger"}
	}
	return p.service
}

func (p servicePreparation) resolve(ctx context.Context, plan pluginBootstrap) (pluginmanager.Lock, error) {
	lock, loadErr := p.cached()
	catalogue := os.Getenv("FCTL_PLUGIN_CATALOGUE")
	if catalogue == "" && loadErr == nil {
		catalogue = lock.Catalogue
	}
	if len(plan.commands) == 1 && plan.commands[0] == p.descriptor().name {
		return lock, loadErr
	}
	if catalogue != "" && !plan.help && len(plan.commands) > 0 && plan.commands[0] == p.descriptor().name {
		return p.matchVersion(ctx, catalogue, nil)
	}
	if catalogue == "" && errors.Is(loadErr, pluginmanager.ErrNotInstalled) && !plan.help &&
		len(plan.commands) > 0 && plan.commands[0] == p.descriptor().name {
		return p.discover(ctx)
	}
	return lock, loadErr
}

func (p servicePreparation) discover(ctx context.Context) (pluginmanager.Lock, error) {
	catalogue, err := p.manager.Discover(ctx, pluginmanager.DefaultCatalogue)
	if errors.Is(err, pluginmanager.ErrCatalogueUnavailable) && ctx.Err() == nil && p.descriptor().name == "ledger" {
		return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
	}
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	for _, release := range catalogue.Releases {
		if release.Service == p.descriptor().name && release.Platform == pluginmanager.CurrentPlatform() {
			return p.matchVersion(ctx, pluginmanager.DefaultCatalogue, &catalogue)
		}
	}
	if p.descriptor().name == "auth" {
		return pluginmanager.Lock{}, fmt.Errorf("%w: Auth has no published plugin for %s; prepare it with fctl plugins sync --service auth or plugins install --service auth --binary PATH", pluginmanager.ErrNoRelease, pluginmanager.CurrentPlatform())
	}
	// Ledger retains its embedded provider until a native release exists.
	return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
}

func (p servicePreparation) cached() (pluginmanager.Lock, error) {
	return cachedServiceLock(p.settings, p.root, p.manager, p.descriptor().name)
}
func (p servicePreparation) matchVersion(ctx context.Context, catalogue string, discovered *pluginmanager.Catalogue) (pluginmanager.Lock, error) {
	target, version, err := pluginServiceSyncTarget(ctx, p.settings, p.root, p.descriptor().name, "")
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	lock, err := p.manager.Load(target, p.descriptor().name)
	notInstalled := errors.Is(err, pluginmanager.ErrNotInstalled)
	if err == nil && lock.ServiceVersion == version {
		return lock, nil
	}
	if err != nil && !errors.Is(err, pluginmanager.ErrNotInstalled) {
		return lock, err
	}
	var release pluginmanager.Release
	if discovered != nil {
		release, err = discovered.Resolve(pluginmanager.CurrentPlatform(), p.descriptor().name, version, 0)
	} else {
		release, err = p.manager.Resolve(ctx, catalogue, p.descriptor().name, version, 0)
	}
	if errors.Is(err, pluginmanager.ErrNoRelease) && discovered != nil && notInstalled && p.descriptor().name == "ledger" {
		return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
	}
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	if err := validateServiceManifest(ctx, p.root, p.descriptor().name, release.Manifest); err != nil {
		return pluginmanager.Lock{}, err
	}
	lock, err = p.manager.InstallResolved(ctx, catalogue, target, release)
	if err != nil {
		return lock, err
	}
	_, err = fmt.Fprintf(p.root.ErrOrStderr(), "Prepared %s plugin %s (revision %d).\n", p.descriptor().title, lock.ServiceVersion, lock.Revision)
	return lock, err
}

func validateLedgerManifest(ctx context.Context, root *cobra.Command, manifest pluginsdk.Manifest) error {
	return validateServiceManifest(ctx, root, "ledger", manifest)
}

//nolint:contextcheck // Validation builds callbacks without executing them.
func validateServiceManifest(ctx context.Context, root *cobra.Command, service string, manifest pluginsdk.Manifest) error {
	if err := serviceManifestIdentity(service, manifest); err != nil {
		return err
	}
	registry := &plugin.Registry{}
	factory := plugin.ExternalFactory("", manifest)
	if err := registry.Register(ctx, factory(nil), factory); err != nil {
		return fmt.Errorf("invalid %s plugin manifest: %w", service, err)
	}
	check := &cobra.Command{Use: "fctl"}
	check.PersistentFlags().AddFlagSet(root.PersistentFlags())
	return plugin.NewCommand(registry, func(context.Context, string) (*api.Client, error) {
		return nil, fmt.Errorf("manifest validation does not execute commands")
	}).AddTo(check)
}
