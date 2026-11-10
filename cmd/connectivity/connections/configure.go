package connections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type ConfigureStore struct {
	Connection connectivity.Connection `json:"connection"`
}

type ConfigureController struct {
	read    configuration.ReadFileFunc
	approve approvalFunc
	store   *ConfigureStore
}

var _ fctl.Controller[*ConfigureStore] = (*ConfigureController)(nil)

func NewConfigureController(read configuration.ReadFileFunc) *ConfigureController {
	return &ConfigureController{
		read:    read,
		approve: fctl.CheckStackApprobation,
		store:   &ConfigureStore{},
	}
}

func NewConfigureCommand(read configuration.ReadFileFunc, paths PathCompleter) *cobra.Command {
	controller := NewConfigureController(read)
	return fctl.NewCommand(
		"configure <connection>",
		fctl.WithAliases("update", "c"),
		fctl.WithShortDescription("Change a connection's version, channel or configuration"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectionNames()),
		fctl.WithStringFlag(versionFlag, "", "Exact connector version to pin, for example v0.5.0-beta.14. Cannot be combined with --channel"),
		fctl.WithStringFlag(channelFlag, "", "Maturity channel to track: stable, rc, beta or alpha. Cannot be combined with --version"),
		fctl.WithStringFlag(pollIntervalFlag, "", "Polling interval as a Go duration, for example 30s. Defaults to the service value"),
		fctl.WithStringFlag(configFlag, "", "Configuration file, YAML or JSON, with an env map and a files map: {env: {KEY: value}, files: {/path/in/plugin: contents}}. a null value removes that key"),
		fctl.WithStringArrayFlag(setFlag, nil, "Configuration value KEY=VALUE, or KEY=@path to read a file (repeatable). Completion follows a --version or --channel typed before it"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*ConfigureStore](controller),
		fctl.WithFlagUsage("confirm", "Skip the confirmation prompt"),
		fctl.WithExamples(
			"fctl connectivity connections configure stripeconnect-fpss --channel beta",
			"fctl connectivity connections configure stripeconnect-fpss \\",
			"    --set HISTORY_START=2026-01-01T00:00:00Z",
		),
		fctl.WithoutFileCompletion(pollIntervalFlag),
		fctl.WithFlagCompletion(versionFlag, completeVersions(resolveConfigureConnector)),
		fctl.WithFlagCompletion(channelFlag, connectivityinternal.CompleteChannels()),
		fctl.WithFlagCompletion(setFlag, CompleteSetValues(resolveConfigureConnectorVersion, paths)),
	)
}

func resolveConfigureConnector(ctx context.Context, client *sdk.FormanceV1, _ *cobra.Command, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	response, err := client.GetConnection(ctx, operations.GetConnectionRequest{Name: url.PathEscape(args[0])}, nil)
	var instance *connectivity.Connection
	if response != nil {
		instance = response.Connection
	}
	if err != nil || instance == nil || instance.Name == "" {
		return "", err
	}
	return fctl.StringPointerToString(instance.ConnectionConfiguration.Connector), nil
}

func resolveConfigureConnectorVersion(ctx context.Context, client *sdk.FormanceV1, cmd *cobra.Command, args []string) (*connectivity.ConnectorVersion, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, nil
	}
	response, err := client.GetConnection(ctx, operations.GetConnectionRequest{Name: url.PathEscape(args[0])}, nil)
	var instance *connectivity.Connection
	if response != nil {
		instance = response.Connection
	}
	if err != nil || instance == nil || instance.Name == "" {
		return nil, err
	}
	connector := fctl.StringPointerToString(instance.ConnectionConfiguration.Connector)
	pinned, channel, err := configureVersionSelector(cmd, instance)
	if err != nil {
		return nil, err
	}
	if channel != "" {
		return resolveChannelVersion(ctx, client, connector, channel, appliedChannelFloor(instance))
	}
	return resolveConnectorVersion(ctx, client, connector, pinned)
}

func configureVersionSelector(cmd *cobra.Command, instance *connectivity.Connection) (string, string, error) {
	pinned, channel := "", ""
	if instance != nil && instance.ConnectionConfiguration.Version != nil && *instance.ConnectionConfiguration.Version != "" {
		pinned = *instance.ConnectionConfiguration.Version
	}
	if instance != nil && instance.ConnectionConfiguration.Channel != nil && *instance.ConnectionConfiguration.Channel != "" {
		channel = string(*instance.ConnectionConfiguration.Channel)
	}
	if cmd.Flags().Changed(versionFlag) {
		return fctl.GetString(cmd, versionFlag), "", nil
	}
	if cmd.Flags().Changed(channelFlag) {
		return "", fctl.GetString(cmd, channelFlag), nil
	}
	if pinned != "" && channel != "" {
		return "", "", fmt.Errorf("connection has conflicting version/channel selectors; specify --version or --channel")
	}
	if pinned != "" {
		return pinned, "", nil
	}
	if channel != "" {
		return "", channel, nil
	}
	return "", "", fmt.Errorf("connection has no version/channel selector; specify --version or --channel")
}

func (c *ConfigureController) GetStore() *ConfigureStore {
	return c.store
}

func (c *ConfigureController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, connectivityinternal.PatchAPI(clients))
}

func (c *ConfigureController) run(cmd *cobra.Command, args []string, client *sdk.FormanceV1) (fctl.Renderable, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}

	name := args[0]
	response, err := client.GetConnection(cmd.Context(), operations.GetConnectionRequest{Name: url.PathEscape(name)}, nil)
	var instance *connectivity.Connection
	if response != nil {
		instance = response.Connection
	}
	if err != nil {
		return nil, err
	}
	if instance == nil || instance.Name == "" {
		return nil, fmt.Errorf("configure connectivity connection %q: empty response", name)
	}

	connector := fctl.StringPointerToString(instance.ConnectionConfiguration.Connector)
	specPatch := map[string]any{}
	configChanged := cmd.Flags().Changed(configFlag) || cmd.Flags().Changed(setFlag)
	if configChanged {
		pinned, channel, err := configureVersionSelector(cmd, instance)
		if err != nil {
			return nil, err
		}
		var version *connectivity.ConnectorVersion
		if channel != "" {
			version, err = resolveChannelVersion(cmd.Context(), client, connector, channel, appliedChannelFloor(instance))
		} else {
			version, err = resolveConnectorVersion(cmd.Context(), client, connector, pinned)
		}
		if err != nil {
			return nil, err
		}
		setValues, err := cmd.Flags().GetStringArray(setFlag)
		if err != nil {
			return nil, err
		}
		intent := map[string]any{}
		_, err = configuration.BuildConfigureConfig(cmd, version, &instance.ConnectionConfiguration, configuration.InputOptions{
			Patch:      intent,
			ConfigFile: fctl.GetString(cmd, configFlag),
			SetValues:  setValues,
		}, c.read)
		if err != nil {
			return nil, err
		}
		for key, value := range intent {
			specPatch[key] = value
		}
	}
	if cmd.Flags().Changed(versionFlag) {
		specPatch["version"] = fctl.GetString(cmd, versionFlag)
		specPatch["channel"] = nil
	}
	if cmd.Flags().Changed(channelFlag) {
		channel := fctl.GetString(cmd, channelFlag)
		specPatch["channel"] = channel
		specPatch["version"] = nil
	}
	if cmd.Flags().Changed(pollIntervalFlag) {
		specPatch["pollInterval"] = fctl.GetString(cmd, pollIntervalFlag)
	}
	if len(specPatch) == 0 {
		return nil, fmt.Errorf("no configuration changes requested")
	}
	if !c.approve(cmd, "You are about to configure Connectivity connection %q", name) {
		return nil, fctl.ErrMissingApproval
	}

	body, err := json.Marshal(map[string]any{"configuration": specPatch})
	if err != nil {
		return nil, err
	}
	responsePatch, err := client.PatchConnection(connectivityinternal.WithPatchBody(cmd.Context(), body), operations.PatchConnectionRequest{Name: url.PathEscape(name)}, nil)
	var updated *connectivity.Connection
	if responsePatch != nil {
		updated = responsePatch.Connection
	}
	if err != nil {
		return nil, err
	}
	if updated == nil || updated.Name == "" {
		return nil, fmt.Errorf("configure connectivity connection %q: empty response", name)
	}
	c.store.Connection = *updated
	return c, nil
}

func (c *ConfigureController) Render(cmd *cobra.Command, _ []string) error {
	return fctl.RenderText(cmd.OutOrStdout(), fmt.Sprintf("Connection %q configured.", c.store.Connection.Name))
}
