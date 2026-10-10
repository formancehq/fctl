package pluginhost

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginselection"
)

// SelectionTarget identifies a provider independently of its artifact cache.
func SelectionTarget(settings *connection.Settings, cmd *cobra.Command, service string) (pluginselection.Target, error) {
	options, _, name, _, err := settings.Resolve(cmd)
	if err != nil {
		return pluginselection.Target{}, err
	}
	endpoint := ""
	switch service {
	case "ledger":
		if options.LedgerURL != "" {
			endpoint = options.LedgerURL
		}
	case "auth":
		if options.AuthURL != "" {
			endpoint = options.AuthURL
		}
	}
	if endpoint == "" && options.StackURL != "" {
		endpoint = strings.TrimRight(options.StackURL, "/") + "/api/" + service
	}
	if options.AuthMode == "cloud" {
		endpoint = options.Issuer
		if endpoint == "" {
			endpoint = "https://app.formance.cloud/api"
		}
	}
	return pluginselection.Target{Profile: name, Organization: options.Organization, Stack: options.Stack, Endpoint: endpoint}, nil
}

func InspectService(ctx context.Context, root *cobra.Command, settings *connection.Settings, service string) (pluginselection.Selection, error) {
	client, err := settings.Client(ctx, root, service)
	if err != nil {
		return pluginselection.Selection{}, err
	}
	data, err := client.Do(ctx, http.MethodGet, "/_info", nil, nil, nil)
	if err != nil {
		return pluginselection.Selection{}, fmt.Errorf("discover %s version: %w", service, err)
	}
	version, err := pluginsdk.ServiceVersion(data)
	if err != nil {
		return pluginselection.Selection{}, err
	}
	line, err := settings.StackVersion(ctx, root)
	if err != nil {
		return pluginselection.Selection{}, err
	}
	target, err := SelectionTarget(settings, root, service)
	if err != nil {
		return pluginselection.Selection{}, err
	}
	return pluginselection.Selection{Target: target, Service: service, ServiceVersion: version, StackVersion: line}, nil
}

// SelectedCommand inspects core flags before a provider supplies its command tree.
func SelectedCommand(root *cobra.Command, args []string) (string, bool, error) {
	plan, err := parsePluginBootstrap(root, args)
	if err != nil {
		return "", false, err
	}
	if len(plan.commands) == 0 {
		return "", true, nil
	}
	if plan.commands[0] == "__complete" || plan.commands[0] == "completion" {
		return "", true, nil
	}
	return plan.commands[0], plan.help || len(plan.commands) == 1, nil
}
