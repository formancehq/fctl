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
	if args == nil {
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
	prep := ledgerPreparation{root: root, settings: settings, manager: manager}
	lock, err := prep.resolve(ctx, plan)
	if errors.Is(err, pluginmanager.ErrNotInstalled) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("prepare Ledger plugin: %w", err)
	}
	if err := validateLedgerManifest(ctx, root, lock.Manifest); err != nil {
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

type ledgerPreparation struct {
	root     *cobra.Command
	settings *connection.Settings
	manager  *pluginmanager.Manager
}

func (p ledgerPreparation) resolve(ctx context.Context, plan pluginBootstrap) (pluginmanager.Lock, error) {
	lock, loadErr := p.cached()
	catalogue := os.Getenv("FCTL_PLUGIN_CATALOGUE")
	if catalogue == "" && loadErr == nil {
		catalogue = lock.Catalogue
	}
	if catalogue != "" && !plan.help && len(plan.commands) > 0 && plan.commands[0] == "ledger" {
		return p.matchVersion(ctx, catalogue)
	}
	return lock, loadErr
}
func (p ledgerPreparation) cached() (pluginmanager.Lock, error) {
	return cachedLedgerLock(p.settings, p.root, p.manager)
}
func (p ledgerPreparation) matchVersion(ctx context.Context, catalogue string) (pluginmanager.Lock, error) {
	target, version, err := pluginSyncTarget(ctx, p.settings, p.root, "")
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	lock, err := p.manager.Load(target, "ledger")
	if err == nil && lock.ServiceVersion == version {
		return lock, nil
	}
	if err != nil && !errors.Is(err, pluginmanager.ErrNotInstalled) {
		return lock, err
	}
	release, err := p.manager.Resolve(ctx, catalogue, "ledger", version, 0)
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	if err := validateLedgerManifest(ctx, p.root, release.Manifest); err != nil {
		return pluginmanager.Lock{}, err
	}
	lock, err = p.manager.InstallResolved(ctx, catalogue, target, release)
	if err != nil {
		return lock, err
	}
	_, err = fmt.Fprintf(p.root.ErrOrStderr(), "Prepared Ledger plugin %s (revision %d).\n", lock.ServiceVersion, lock.Revision)
	return lock, err
}

//nolint:contextcheck // Validation builds callbacks without executing them.
func validateLedgerManifest(ctx context.Context, root *cobra.Command, manifest pluginsdk.Manifest) error {
	if err := ledgerManifestIdentity(manifest); err != nil {
		return err
	}
	registry := &plugin.Registry{}
	factory := plugin.ExternalFactory("", manifest)
	if err := registry.Register(ctx, factory(nil), factory); err != nil {
		return fmt.Errorf("invalid Ledger plugin manifest: %w", err)
	}
	check := &cobra.Command{Use: "fctl"}
	check.PersistentFlags().AddFlagSet(root.PersistentFlags())
	return plugin.NewCommand(registry, func(context.Context, string) (*api.Client, error) {
		return nil, fmt.Errorf("manifest validation does not execute commands")
	}).AddTo(check)
}
