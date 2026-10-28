package connections

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connectors"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	nameFlag         = "name"
	ledgerFlag       = "ledger"
	versionFlag      = "version"
	channelFlag      = "channel"
	pollIntervalFlag = "poll-interval"
	configFlag       = "config"
	envFileFlag      = "env-file"
	setFlag          = "set"
)

type InstallStore struct {
	Connection connectivityclient.Connection `json:"connection"`
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
		"install <connector>",
		fctl.WithAliases("create", "in"),
		fctl.WithShortDescription("Install a Connectivity connection"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(connectors.CompleteConnectorNames(factory)),
		fctl.WithStringFlag(nameFlag, "", "Connection name (defaults to the connector name)"),
		fctl.WithStringFlag(ledgerFlag, "", "Ledger name"),
		fctl.WithStringFlag(versionFlag, "", "Connector version"),
		fctl.WithStringFlag(channelFlag, "", "Maturity channel to track (stable, rc, beta, alpha; defaults to stable when no version is pinned)"),
		fctl.WithStringFlag(initialCursorFlag, "", "Initial ingestion cursor as base64 (opaque; does not reset a running cursor)"),
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
	if err := command.RegisterFlagCompletionFunc(versionFlag, CompleteVersions(factory, installConnectorArgument)); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(channelFlag, connectivityinternal.CompleteChannels(factory)); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(setFlag, CompleteSetValues(factory, resolveInstallConnectorVersion, paths)); err != nil {
		panic(err)
	}
	return command
}

func installConnectorArgument(_ *cobra.Command, args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func resolveInstallConnectorVersion(ctx context.Context, client connectivityclient.Client, cmd *cobra.Command, args []string) (*connectivityclient.ConnectorVersion, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, nil
	}
	return resolveConnectorVersion(ctx, client, args[0], installVersionSelector(cmd))
}

// installVersionSelector picks the {version} slot value governing the config
// schema: the explicit pin, else the tracked channel alias (resolved with
// exactly the rules installation uses), else the CLI's explicit stable channel.
func installVersionSelector(cmd *cobra.Command) string {
	if pinned := fctl.GetString(cmd, versionFlag); pinned != "" {
		return pinned
	}
	if channel := fctl.GetString(cmd, channelFlag); channel != "" {
		return channel
	}
	return "stable"
}

func (c *InstallController) GetStore() *InstallStore {
	return c.store
}

func (c *InstallController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}
	cursor, err := initialCursor(cmd)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(fctl.GetString(cmd, ledgerFlag)) == "" {
		return nil, fmt.Errorf("--ledger cannot be empty")
	}
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}

	connectorName := args[0]
	version, err := resolveConnectorVersion(cmd.Context(), client, connectorName, installVersionSelector(cmd))
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
	config, err := BuildInstallConfig(cmd, version, InputOptions{
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
		name = connectorName
	}
	spec := connectivityclient.ConnectionConfiguration{
		Connector:     connectorName,
		Ledger:        fctl.GetString(cmd, ledgerFlag),
		InitialCursor: cursor,
	}
	if config != nil {
		spec.Env, spec.Files = config.Env, config.Files
	}
	if cmd.Flags().Changed(versionFlag) {
		value, err := cmd.Flags().GetString(versionFlag)
		if err != nil {
			return nil, err
		}
		spec.Version = fctl.Ptr(value)
	}
	if spec.Version == nil {
		spec.Channel = fctl.Ptr(installVersionSelector(cmd))
	}
	if cmd.Flags().Changed(pollIntervalFlag) {
		value, err := cmd.Flags().GetString(pollIntervalFlag)
		if err != nil {
			return nil, err
		}
		spec.PollInterval = fctl.Ptr(value)
	}

	if !c.approve(cmd, "You are about to install Connectivity connection %q", name) {
		return nil, fctl.ErrMissingApproval
	}
	instance, err := client.CreateConnection(cmd.Context(), connectivityclient.ConnectionCreate{Name: name, Configuration: spec})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("install connectivity connection %q: empty response", name)
	}
	c.store.Connection = *instance
	return c, nil
}

func configHasData(config *connectivityclient.ConnectionConfig) bool {
	return config != nil && (len(config.Env) > 0 || len(config.Files) > 0)
}

func (c *InstallController) Render(cmd *cobra.Command, _ []string) error {
	instance := c.store.Connection
	_, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"Connection %q installed with connector %q for ledger %q.\n",
		instance.Name,
		instance.Configuration.ConnectorName(),
		instance.Configuration.Ledger,
	)
	return err
}
