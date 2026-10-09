package cmd

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func ledgerTarget(settings *connection.Settings, cmd *cobra.Command) (pluginmanager.Target, error) {
	options, _, name, _, err := settings.Resolve(cmd)
	if err != nil {
		return pluginmanager.Target{}, err
	}
	endpoint := options.LedgerURL
	if options.AuthMode == "cloud" {
		endpoint = cmp.Or(options.Issuer, cloud.DefaultIssuer)
	} else if endpoint == "" && options.StackURL != "" {
		endpoint = strings.TrimRight(options.StackURL, "/") + "/api/ledger"
	}
	if endpoint == "" {
		return pluginmanager.Target{}, fmt.Errorf("select a profile or configure --ledger-url or --stack-url")
	}
	return pluginmanager.Target{Profile: name, Organization: options.Organization, Stack: options.Stack, Endpoint: endpoint}, nil
}

// A sole prepared Cloud target supplies offline metadata when profile options
// omit organization or stack. It does not select the execution target: the
// authenticated connection still resolves that from verified Cloud claims.
func cachedLedgerLock(settings *connection.Settings, cmd *cobra.Command, manager *pluginmanager.Manager) (pluginmanager.Lock, error) {
	target, err := ledgerTarget(settings, cmd)
	if err != nil {
		return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
	}
	lock, err := manager.Load(target, "ledger")
	if !errors.Is(err, pluginmanager.ErrNotInstalled) || (target.Organization != "" && target.Stack != "") {
		return lock, err
	}
	options, _, _, _, err := settings.Resolve(cmd)
	if err != nil || options.AuthMode != "cloud" {
		return pluginmanager.Lock{}, pluginmanager.ErrNotInstalled
	}
	return soleLedgerLock(manager, target)
}

func soleLedgerLock(manager *pluginmanager.Manager, target pluginmanager.Target) (pluginmanager.Lock, error) {
	locks, err := manager.List()
	if err != nil {
		return pluginmanager.Lock{}, err
	}
	var selected pluginmanager.Lock
	for _, lock := range locks {
		if lock.Service != "ledger" || lock.Target.Profile != target.Profile || lock.Target.Endpoint != target.Endpoint {
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

func ledgerServiceVersion(ctx context.Context, client *api.Client) (string, error) {
	data, err := client.Do(ctx, "GET", "/_info", nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("discover Ledger version: %w", err)
	}
	var info struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return "", fmt.Errorf("decode Ledger version: %w", err)
	}
	version := strings.TrimPrefix(info.Version, "v")
	if version == "" {
		return "", fmt.Errorf("ledger /_info did not report a version")
	}
	return version, nil
}

func serviceCommand(cmd *cobra.Command) string {
	for cmd.Parent() != nil && cmd.Parent().Parent() != nil {
		cmd = cmd.Parent()
	}
	return cmd.Name()
}
