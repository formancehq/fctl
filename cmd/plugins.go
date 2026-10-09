package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func newPluginsCommand(settings *connection.Settings) *cobra.Command {
	root := &cobra.Command{Use: "plugins", Short: "Prepare and inspect external service plugins"}
	root.AddCommand(newPluginsListCommand(settings), newPluginsShowCommand(settings), newPluginsSyncCommand(settings), newPluginsInstallCommand(settings))
	return root
}

func pluginManager(settings *connection.Settings, cmd *cobra.Command) (*pluginmanager.Manager, error) {
	dir, err := settings.DirectoryPath()
	if err != nil {
		return nil, err
	}
	client := settings.HTTPClient(cmd.ErrOrStderr())
	client.CheckRedirect = nil // The manager validates public download redirects.
	return pluginmanager.New(filepath.Join(dir, "plugins"), client)
}

func writePluginResult(cmd *cobra.Command, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return command.WriteJSON(cmd.OutOrStdout(), data)
}

func pluginSyncTarget(ctx context.Context, settings *connection.Settings, cmd *cobra.Command, version string) (pluginmanager.Target, string, error) {
	if version == "" {
		client, err := settings.Client(ctx, cmd, "ledger")
		if err != nil {
			return pluginmanager.Target{}, "", err
		}
		version, err = ledgerServiceVersion(ctx, client)
		if err != nil {
			return pluginmanager.Target{}, "", err
		}
	}
	target, err := ledgerTarget(settings, cmd)
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

type pluginSummary struct {
	Profile      string `json:"profile"`
	Organization string `json:"organization,omitzero"`
	Stack        string `json:"stack,omitzero"`
	Endpoint     string `json:"endpoint"`
	Service      string `json:"service"`
	Version      string `json:"version"`
	Revision     int    `json:"revision"`
	Platform     string `json:"platform"`
	Source       string `json:"source"`
	SHA256       string `json:"sha256"`
}

func summarizePlugin(lock pluginmanager.Lock) pluginSummary {
	source := lock.ArtifactDigest
	if lock.Local {
		source = "local"
	}
	return pluginSummary{Profile: lock.Target.Profile, Organization: lock.Target.Organization, Stack: lock.Target.Stack,
		Endpoint: lock.Target.Endpoint, Service: lock.Service, Version: lock.ServiceVersion, Revision: lock.Revision,
		Platform: lock.Platform.String(), Source: source, SHA256: lock.SHA256}
}

// Keep plugin identities explicit during the Ledger-only pilot.
func ledgerManifestIdentity(manifest pluginsdk.Manifest) error {
	if manifest.Name != "ledger" || manifest.Service != "ledger" || pluginsdk.CommandName(manifest.Root) != "ledger" {
		return fmt.Errorf("this pilot accepts Ledger plugins with the ledger command root")
	}
	return nil
}
