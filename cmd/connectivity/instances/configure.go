package instances

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ConfigureStore struct {
	Instance connectivityclient.Instance `json:"instance"`
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
		"configure <instance>",
		fctl.WithAliases("config", "update", "c"),
		fctl.WithShortDescription("Configure a Connectivity plugin instance"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteInstanceNames(factory)),
		fctl.WithStringFlag(ledgerFlag, "", "Ledger name"),
		fctl.WithStringFlag(versionFlag, "", "Plugin version"),
		withInt64Flag(startSequenceFlag, 0, "Starting ledger sequence"),
		fctl.WithStringFlag(pollIntervalFlag, "", "Polling interval"),
		fctl.WithStringFlag(configFlag, "", "YAML or JSON configuration file"),
		fctl.WithStringArrayFlag(envFileFlag, nil, "Dotenv configuration file (repeatable)"),
		fctl.WithStringArrayFlag(setFlag, nil, "Configuration value KEY=VALUE (repeatable)"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*ConfigureStore](controller),
	)
	if err := command.RegisterFlagCompletionFunc(versionFlag, completeVersions(factory, resolveConfigurePlugin)); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(setFlag, CompleteSetValues(factory, resolveConfigurePlugin, paths)); err != nil {
		panic(err)
	}
	return command
}

func resolveConfigurePlugin(ctx context.Context, client connectivityclient.Client, _ *cobra.Command, args []string) (*connectivityclient.Plugin, error) {
	if len(args) == 0 {
		return nil, nil
	}
	instance, err := client.GetInstance(ctx, args[0])
	if err != nil || instance == nil {
		return nil, err
	}
	return client.GetPlugin(ctx, instance.Spec.Plugin)
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
	instance, err := client.GetInstance(cmd.Context(), name)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("configure connectivity instance %q: empty instance response", name)
	}
	plugin, err := client.GetPlugin(cmd.Context(), instance.Spec.Plugin)
	if err != nil {
		return nil, err
	}
	if plugin == nil {
		return nil, fmt.Errorf("configure connectivity instance %q: empty plugin response", name)
	}

	specPatch := map[string]any{}
	configChanged := cmd.Flags().Changed(configFlag) || cmd.Flags().Changed(envFileFlag) || cmd.Flags().Changed(setFlag)
	if configChanged {
		envFiles, err := cmd.Flags().GetStringArray(envFileFlag)
		if err != nil {
			return nil, err
		}
		setValues, err := cmd.Flags().GetStringArray(setFlag)
		if err != nil {
			return nil, err
		}
		config, err := BuildConfigureConfig(cmd, plugin, instance.Spec.Config, InputOptions{
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
	if cmd.Flags().Changed(startSequenceFlag) {
		startSequence, err := cmd.Flags().GetInt64(startSequenceFlag)
		if err != nil {
			return nil, err
		}
		specPatch["startSequence"] = startSequence
	}
	if cmd.Flags().Changed(pollIntervalFlag) {
		specPatch["pollInterval"] = fctl.GetString(cmd, pollIntervalFlag)
	}
	if len(specPatch) == 0 {
		return nil, fmt.Errorf("no configuration changes requested")
	}
	if !c.approve(cmd, "You are about to configure Connectivity instance %q", name) {
		return nil, fctl.ErrMissingApproval
	}

	updated, err := client.PatchInstance(cmd.Context(), name, connectivityclient.InstancePatch{"spec": specPatch})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, fmt.Errorf("configure connectivity instance %q: empty instance response", name)
	}
	c.store.Instance = *updated
	return c, nil
}

func (c *ConfigureController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Instance %q configured.\n", stringValue(c.store.Instance.Metadata.Name))
	return err
}
