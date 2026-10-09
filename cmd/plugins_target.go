package cmd

import (
	"cmp"
	"context"
	"encoding/json"
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
