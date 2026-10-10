package pluginhost

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

type serviceDescriptor struct {
	name  string
	title string
}

func supportedPluginService(service string) (serviceDescriptor, error) {
	switch service {
	case "auth":
		return serviceDescriptor{name: "auth", title: "Auth"}, nil
	case "ledger":
		return serviceDescriptor{name: "ledger", title: "Ledger"}, nil
	default:
		return serviceDescriptor{}, fmt.Errorf("unsupported external plugin service %q", service)
	}
}

func pluginServiceTarget(settings *connection.Settings, cmd *cobra.Command, service string) (pluginmanager.Target, error) {
	descriptor, err := supportedPluginService(service)
	if err != nil {
		return pluginmanager.Target{}, err
	}
	options, _, name, _, err := settings.Resolve(cmd)
	if err != nil {
		return pluginmanager.Target{}, err
	}
	endpoint := options.LedgerURL
	if descriptor.name == "auth" {
		endpoint = options.AuthURL
	}
	if options.AuthMode == "cloud" {
		endpoint = cmp.Or(options.Issuer, cloud.DefaultIssuer)
	} else if endpoint == "" && options.StackURL != "" {
		endpoint = strings.TrimRight(options.StackURL, "/") + "/api/" + descriptor.name
	}
	if endpoint == "" {
		return pluginmanager.Target{}, fmt.Errorf("select a profile or configure --%s-url or --stack-url", descriptor.name)
	}
	return pluginmanager.Target{Profile: name, Organization: options.Organization, Stack: options.Stack, Endpoint: endpoint}, nil
}

// A sole prepared Cloud target supplies offline metadata when profile options
// omit organization or stack. It does not select the execution target: the
// authenticated connection still resolves that from verified Cloud claims.
func cachedServiceLock(settings *connection.Settings, cmd *cobra.Command, manager *pluginmanager.Manager, service string) (pluginmanager.Lock, error) {
	if _, err := supportedPluginService(service); err != nil {
		return pluginmanager.Lock{}, err
	}
	target, err := pluginServiceTarget(settings, cmd, service)
	if err != nil {
		return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
	}
	lock, err := manager.Load(target, service)
	if !errors.Is(err, pluginmanager.ErrNotInstalled) || (target.Organization != "" && target.Stack != "") {
		return lock, err
	}
	options, _, _, _, err := settings.Resolve(cmd)
	if err != nil || options.AuthMode != "cloud" {
		return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
	}
	return soleServiceLock(manager, target, service)
}

func soleServiceLock(manager *pluginmanager.Manager, target pluginmanager.Target, service string) (pluginmanager.Lock, error) {
	locks, err := manager.List()
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	var selected pluginmanager.Lock
	for _, lock := range locks {
		if lock.Service != service || lock.Target.Profile != target.Profile || lock.Target.Endpoint != target.Endpoint {
			continue
		}
		if (target.Organization != "" && lock.Target.Organization != target.Organization) || (target.Stack != "" && lock.Target.Stack != target.Stack) {
			continue
		}
		if selected.Service != "" {
			return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
		}
		selected = lock
	}
	if selected.Service == "" {
		return selected, pluginmanager.ErrNotInstalled
	}
	return selected, nil
}

func pluginServiceVersion(ctx context.Context, client *api.Client, service string) (string, error) {
	descriptor, err := supportedPluginService(service)
	if err != nil {
		return "", err
	}
	data, err := client.Do(ctx, "GET", "/_info", nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("discover %s version: %w", descriptor.title, err)
	}
	version, err := pluginsdk.ServiceVersion(data)
	if err != nil {
		return "", fmt.Errorf("decode %s version: %w", descriptor.title, err)
	}
	return version, nil
}

// ServiceCommand identifies the top-level service of a Cobra command.
func ServiceCommand(cmd *cobra.Command) string {
	for cmd.Parent() != nil && cmd.Parent().Parent() != nil {
		cmd = cmd.Parent()
	}
	return cmd.Name()
}
