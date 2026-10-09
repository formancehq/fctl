package plugins

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func writePluginResult(cmd *cobra.Command, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return command.WriteJSON(cmd.OutOrStdout(), data)
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
