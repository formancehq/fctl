package instances

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	"github.com/formancehq/fctl/v3/cmd/connectivity/plugins"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	nameFlag          = "name"
	ledgerFlag        = "ledger"
	versionFlag       = "version"
	startSequenceFlag = "start-sequence"
	pollIntervalFlag  = "poll-interval"
	configFlag        = "config"
	envFileFlag       = "env-file"
	setFlag           = "set"
)

type InstallStore struct {
	Instance connectivityclient.Instance `json:"instance"`
}

type approvalFunc func(*cobra.Command, string, ...any) bool

type InstallController struct {
	factory connectivityinternal.ClientFactory
	read    ReadFileFunc
	approve approvalFunc
	store   *InstallStore
}

var _ fctl.Controller[*InstallStore] = (*InstallController)(nil)

func NewInstallController(factory connectivityinternal.ClientFactory, read ReadFileFunc) *InstallController {
	return &InstallController{
		factory: factory,
		read:    read,
		approve: fctl.CheckStackApprobation,
		store:   &InstallStore{},
	}
}

func NewInstallCommand(factory connectivityinternal.ClientFactory, read ReadFileFunc, paths PathCompleter) *cobra.Command {
	controller := NewInstallController(factory, read)
	command := fctl.NewCommand(
		"install <plugin>",
		fctl.WithAliases("create", "in"),
		fctl.WithShortDescription("Install a Connectivity plugin instance"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(plugins.CompletePluginNames(factory)),
		fctl.WithStringFlag(nameFlag, "", "Instance name (defaults to the plugin name)"),
		fctl.WithStringFlag(ledgerFlag, "", "Ledger name"),
		fctl.WithStringFlag(versionFlag, "", "Plugin version"),
		withInt64Flag(startSequenceFlag, 0, "Starting ledger sequence"),
		fctl.WithStringFlag(pollIntervalFlag, "", "Polling interval"),
		fctl.WithStringFlag(configFlag, "", "YAML or JSON configuration file"),
		fctl.WithStringArrayFlag(envFileFlag, nil, "Dotenv configuration file (repeatable)"),
		fctl.WithStringArrayFlag(setFlag, nil, "Configuration value KEY=VALUE (repeatable)"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*InstallStore](controller),
	)
	if err := command.MarkFlagRequired(ledgerFlag); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(versionFlag, CompleteVersions(factory, installPluginArgument)); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(setFlag, CompleteSetValues(factory, resolveInstallPlugin, paths)); err != nil {
		panic(err)
	}
	return command
}

func withInt64Flag(name string, defaultValue int64, help string) fctl.CommandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().Int64(name, defaultValue, help)
	}
}

func installPluginArgument(_ *cobra.Command, args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func resolveInstallPlugin(ctx context.Context, client connectivityclient.Client, _ *cobra.Command, args []string) (*connectivityclient.Plugin, error) {
	if len(args) == 0 {
		return nil, nil
	}
	return client.GetPlugin(ctx, args[0])
}

func (c *InstallController) GetStore() *InstallStore {
	return c.store
}

func (c *InstallController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}

	pluginName := args[0]
	plugin, err := client.GetPlugin(cmd.Context(), pluginName)
	if err != nil {
		return nil, err
	}
	if plugin == nil {
		return nil, fmt.Errorf("install connectivity plugin %q: empty plugin response", pluginName)
	}

	envFiles, err := cmd.Flags().GetStringArray(envFileFlag)
	if err != nil {
		return nil, err
	}
	setValues, err := cmd.Flags().GetStringArray(setFlag)
	if err != nil {
		return nil, err
	}
	config, err := BuildInstallConfig(cmd, plugin, InputOptions{
		ConfigFile: fctl.GetString(cmd, configFlag),
		EnvFiles:   envFiles,
		SetValues:  setValues,
	}, c.read)
	if err != nil {
		return nil, err
	}
	if !configHasData(config) {
		config = nil
	}

	name := fctl.GetString(cmd, nameFlag)
	if name == "" {
		name = pluginName
	}
	spec := connectivityclient.InstanceSpec{
		Plugin: pluginName,
		Ledger: fctl.GetString(cmd, ledgerFlag),
		Config: config,
	}
	if cmd.Flags().Changed(versionFlag) {
		value, err := cmd.Flags().GetString(versionFlag)
		if err != nil {
			return nil, err
		}
		spec.Version = fctl.Ptr(value)
	}
	if cmd.Flags().Changed(startSequenceFlag) {
		value, err := cmd.Flags().GetInt64(startSequenceFlag)
		if err != nil {
			return nil, err
		}
		spec.StartSequence = fctl.Ptr(value)
	}
	if cmd.Flags().Changed(pollIntervalFlag) {
		value, err := cmd.Flags().GetString(pollIntervalFlag)
		if err != nil {
			return nil, err
		}
		spec.PollInterval = fctl.Ptr(value)
	}

	if !c.approve(cmd, "You are about to install Connectivity instance %q", name) {
		return nil, fctl.ErrMissingApproval
	}
	instance, err := client.CreateInstance(cmd.Context(), connectivityclient.InstanceCreate{Name: name, Spec: spec})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("install connectivity instance %q: empty instance response", name)
	}
	c.store.Instance = *instance
	return c, nil
}

func configHasData(config *connectivityclient.InstanceConfig) bool {
	return config != nil && (len(config.Env) > 0 || len(config.Files) > 0)
}

func (c *InstallController) Render(cmd *cobra.Command, _ []string) error {
	instance := c.store.Instance
	_, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"Instance %q installed with plugin %q for ledger %q.\n",
		stringValue(instance.Metadata.Name),
		instance.Spec.Plugin,
		instance.Spec.Ledger,
	)
	return err
}
