// Package pluginhost prepares external service providers and manages their target caches.
package pluginhost

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func pluginManager(settings *connection.Settings, cmd *cobra.Command) (*pluginmanager.Manager, error) {
	dir, err := settings.DirectoryPath()
	if err != nil {
		return nil, err
	}
	client := settings.HTTPClient(cmd.ErrOrStderr())
	client.CheckRedirect = nil // The manager validates public download redirects.
	return pluginmanager.New(filepath.Join(dir, "plugins"), client)
}

func pluginServiceSyncTarget(ctx context.Context, settings *connection.Settings, cmd *cobra.Command, service, version string) (pluginmanager.Target, string, error) {
	if _, err := supportedPluginService(service); err != nil {
		return pluginmanager.Target{}, "", err
	}
	if version == "" {
		client, err := settings.Client(ctx, cmd, service)
		if err != nil {
			return pluginmanager.Target{}, "", err
		}
		version, err = pluginServiceVersion(ctx, client, service)
		if err != nil {
			return pluginmanager.Target{}, "", err
		}
	}
	target, err := pluginServiceTarget(settings, cmd, service)
	if err != nil {
		return target, "", err
	}
	options, _, _, _, err := settings.Resolve(cmd)
	if err != nil {
		return target, "", err
	}
	if err := connection.Validate(options); err != nil {
		return target, "", err
	}
	if options.AuthMode == "cloud" && (target.Organization == "" || target.Stack == "") {
		err = fmt.Errorf("choose --organization and --stack, or omit --service-version to select the Cloud target")
	}
	return target, strings.TrimPrefix(version, "v"), err
}
