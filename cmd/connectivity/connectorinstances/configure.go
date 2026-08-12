package connectorinstances

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ConfigureStore struct {
	ConnectorInstance connectivityclient.ConnectorInstance `json:"connectorInstance"`
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
		"configure <connectorinstance>",
		fctl.WithAliases("config", "update", "c"),
		fctl.WithShortDescription("Configure a Connectivity connector instance"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorInstanceNames(factory)),
		fctl.WithStringFlag(ledgerFlag, "", "Ledger name"),
		fctl.WithStringFlag(versionFlag, "", "Connector version"),
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
	if err := command.RegisterFlagCompletionFunc(setFlag, CompleteSetValues(factory, resolveConfigureConnectorVersion, paths)); err != nil {
		panic(err)
	}
	return command
}

func resolveConfigureConnector(ctx context.Context, client connectivityclient.Client, _ *cobra.Command, args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	instance, err := client.GetConnectorInstance(ctx, args[0])
	if err != nil || instance == nil {
		return "", err
	}
	return instance.Spec.Connector, nil
}

func resolveConfigureConnectorVersion(ctx context.Context, client connectivityclient.Client, cmd *cobra.Command, args []string) (*connectivityclient.ConnectorVersion, error) {
	if len(args) == 0 {
		return nil, nil
	}
	instance, err := client.GetConnectorInstance(ctx, args[0])
	if err != nil || instance == nil {
		return nil, err
	}
	pinned := fctl.GetString(cmd, versionFlag)
	if pinned == "" {
		pinned = instanceVersionPin(instance)
	}
	return resolveConnectorVersion(ctx, client, instance.Spec.Connector, pinned)
}

func (c *ConfigureController) GetStore() *ConfigureStore {
	return c.store
}

func (c *ConfigureController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}

	name := args[0]
	instance, err := client.GetConnectorInstance(cmd.Context(), name)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("configure connectivity connector instance %q: empty response", name)
	}

	specPatch := map[string]any{}
	configChanged := cmd.Flags().Changed(configFlag) || cmd.Flags().Changed(envFileFlag) || cmd.Flags().Changed(setFlag)
	if configChanged {
		pinned := instanceVersionPin(instance)
		if cmd.Flags().Changed(versionFlag) {
			pinned = fctl.GetString(cmd, versionFlag)
		}
		version, err := resolveConnectorVersion(cmd.Context(), client, instance.Spec.Connector, pinned)
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
		config, err := BuildConfigureConfig(cmd, version, instance.Spec.Config, InputOptions{
			ConfigFile: fctl.GetString(cmd, configFlag),
			EnvFiles:   envFiles,
			SetValues:  setValues,
		}, c.read)
		if err != nil {
			return nil, err
		}
		specPatch["config"] = config
	}
	if cmd.Flags().Changed(versionFlag) {
		specPatch["version"] = fctl.GetString(cmd, versionFlag)
	}
	if cmd.Flags().Changed(ledgerFlag) {
		specPatch["ledger"] = fctl.GetString(cmd, ledgerFlag)
	}
	if cmd.Flags().Changed(pollIntervalFlag) {
		specPatch["pollInterval"] = fctl.GetString(cmd, pollIntervalFlag)
	}
	if len(specPatch) == 0 {
		return nil, fmt.Errorf("no configuration changes requested")
	}
	if !c.approve(cmd, "You are about to configure Connectivity connector instance %q", name) {
		return nil, fctl.ErrMissingApproval
	}

	updated, err := client.PatchConnectorInstance(cmd.Context(), name, connectivityclient.ConnectorInstancePatch{"spec": specPatch})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, fmt.Errorf("configure connectivity connector instance %q: empty response", name)
	}
	c.store.ConnectorInstance = *updated
	return c, nil
}

func (c *ConfigureController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Connector instance %q configured.\n", stringValue(c.store.ConnectorInstance.Metadata.Name))
	return err
}
