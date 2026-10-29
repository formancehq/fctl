package connections

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ConfigureStore struct {
	Connection connectivityclient.Connection `json:"connection"`
}

type ConfigureController struct {
	factory connectivityinternal.ClientFactory
	read    ReadFileFunc
	approve approvalFunc
	store   *ConfigureStore
}

var _ fctl.Controller[*ConfigureStore] = (*ConfigureController)(nil)

func NewConfigureController(factory connectivityinternal.ClientFactory, read ReadFileFunc) *ConfigureController {
	return &ConfigureController{
		factory: factory,
		read:    read,
		approve: fctl.CheckStackApprobation,
		store:   &ConfigureStore{},
	}
}

func NewConfigureCommand(factory connectivityinternal.ClientFactory, read ReadFileFunc, paths PathCompleter) *cobra.Command {
	controller := NewConfigureController(factory, read)
	command := fctl.NewCommand(
		"configure <connection>",
		fctl.WithAliases("config", "update", "c"),
		fctl.WithShortDescription("Configure a Connectivity connection"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectionNames(factory)),
		fctl.WithStringFlag(versionFlag, "", "Connector version"),
		fctl.WithStringFlag(channelFlag, "", "Maturity channel to track (stable, rc, beta, alpha) when no version is pinned"),
		fctl.WithStringFlag(initialCursorFlag, "", "Initial ingestion cursor as base64 (opaque; does not reset a running cursor)"),
		fctl.WithStringFlag(pollIntervalFlag, "", "Polling interval"),
		fctl.WithStringFlag(configFlag, "", "YAML or JSON configuration file"),
		fctl.WithStringArrayFlag(envFileFlag, nil, "Dotenv configuration file (repeatable)"),
		fctl.WithStringArrayFlag(setFlag, nil, "Configuration value KEY=VALUE (repeatable)"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*ConfigureStore](controller),
	)
	if err := command.RegisterFlagCompletionFunc(versionFlag, completeVersions(factory, resolveConfigureConnector)); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(channelFlag, connectivityinternal.CompleteChannels(factory)); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(setFlag, CompleteSetValues(factory, resolveConfigureConnectorVersion, paths)); err != nil {
		panic(err)
	}
	return command
}

func resolveConfigureConnector(ctx context.Context, client connectivityclient.Client, _ *cobra.Command, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	instance, err := client.GetConnection(ctx, args[0])
	if err != nil || instance == nil {
		return "", err
	}
	return instance.Configuration.ConnectorName(), nil
}

func resolveConfigureConnectorVersion(ctx context.Context, client connectivityclient.Client, cmd *cobra.Command, args []string) (*connectivityclient.ConnectorVersion, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, nil
	}
	instance, err := client.GetConnection(ctx, args[0])
	if err != nil || instance == nil {
		return nil, err
	}
	connector := instance.Configuration.ConnectorName()
	pinned, channel, err := configureVersionSelector(cmd, instance)
	if err != nil {
		return nil, err
	}
	if channel != "" {
		return resolveChannelVersion(ctx, client, connector, channel, appliedChannelFloor(instance))
	}
	return resolveConnectorVersion(ctx, client, connector, pinned)
}

func configureVersionSelector(cmd *cobra.Command, instance *connectivityclient.Connection) (string, string, error) {
	pinned, channel := "", ""
	if instance != nil && instance.Configuration.Version != nil && *instance.Configuration.Version != "" {
		pinned = *instance.Configuration.Version
	}
	if instance != nil && instance.Configuration.Channel != nil && *instance.Configuration.Channel != "" {
		channel = *instance.Configuration.Channel
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
	cursor, err := initialCursor(cmd)
	if err != nil {
		return nil, err
	}
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}

	name := args[0]
	instance, err := client.GetConnection(cmd.Context(), name)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("configure connectivity connection %q: empty response", name)
	}

	connector := instance.Configuration.ConnectorName()
	specPatch := map[string]any{}
	configChanged := cmd.Flags().Changed(configFlag) || cmd.Flags().Changed(envFileFlag) || cmd.Flags().Changed(setFlag)
	if configChanged {
		pinned, channel, err := configureVersionSelector(cmd, instance)
		if err != nil {
			return nil, err
		}
		var version *connectivityclient.ConnectorVersion
		if channel != "" {
			version, err = resolveChannelVersion(cmd.Context(), client, connector, channel, appliedChannelFloor(instance))
		} else {
			version, err = resolveConnectorVersion(cmd.Context(), client, connector, pinned)
		}
		if err != nil {
			return nil, err
		}
		envFiles, err := cmd.Flags().GetStringArray(envFileFlag)
		if err != nil {
			return nil, err
		}
		setValues, err := cmd.Flags().GetStringArray(setFlag)
		if err != nil {
			return nil, err
		}
		intent := map[string]any{}
		_, err = BuildConfigureConfig(cmd, version, instance.Configuration.Configuration(), InputOptions{
			patch:      intent,
			ConfigFile: fctl.GetString(cmd, configFlag),
			EnvFiles:   envFiles,
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
	if cmd.Flags().Changed(initialCursorFlag) {
		specPatch["initialCursor"] = cursor
	}
	if len(specPatch) == 0 {
		return nil, fmt.Errorf("no configuration changes requested")
	}
	if !c.approve(cmd, "You are about to configure Connectivity connection %q", name) {
		return nil, fctl.ErrMissingApproval
	}

	updated, err := client.PatchConnection(cmd.Context(), name, connectivityclient.ConnectionPatch{"configuration": specPatch})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, fmt.Errorf("configure connectivity connection %q: empty response", name)
	}
	c.store.Connection = *updated
	return c, nil
}

func (c *ConfigureController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Connection %q configured.\n", c.store.Connection.Name)
	return err
}
