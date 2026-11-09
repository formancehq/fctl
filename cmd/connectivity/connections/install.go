package connections

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	"github.com/formancehq/fctl/v3/cmd/connectivity/connectors"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

const (
	initialCursorFlag = "initial-cursor"
	nameFlag          = "name"
	ledgerFlag        = "ledger"
	versionFlag       = "version"
	channelFlag       = "channel"
	pollIntervalFlag  = "poll-interval"
	configFlag        = "config"
	envFileFlag       = "env-file"
	setFlag           = "set"
)

type InstallStore struct {
	Connection connectivity.Connection `json:"connection"`
}

type approvalFunc func(*cobra.Command, string, ...any) bool

type InstallController struct {
	read    configuration.ReadFileFunc
	approve approvalFunc
	store   *InstallStore
}

var _ fctl.Controller[*InstallStore] = (*InstallController)(nil)

func NewInstallController(read configuration.ReadFileFunc) *InstallController {
	return &InstallController{
		read:    read,
		approve: fctl.CheckStackApprobation,
		store:   &InstallStore{},
	}
}

func NewInstallCommand(read configuration.ReadFileFunc, paths PathCompleter) *cobra.Command {
	controller := NewInstallController(read)
	return fctl.NewCommand(
		"install <connector>",
		fctl.WithAliases("create", "in"),
		fctl.WithShortDescription("Install a connection from a connector"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(connectors.CompleteConnectorNames()),
		fctl.WithStringFlag(nameFlag, "", "Connection name. Defaults to the connector name"),
		fctl.WithStringFlag(ledgerFlag, "", "(required) Target ledger. Immutable once set"),
		fctl.WithStringFlag(versionFlag, "", "Exact connector version to pin, for example v0.5.0-beta.14. Cannot be combined with --channel"),
		fctl.WithStringFlag(channelFlag, "", "Maturity channel to track: stable, rc, beta or alpha. Defaults to stable when --version is not set"),
		fctl.WithStringFlag(initialCursorFlag, "", "Base64 cursor ingestion starts from. Does not reset a running cursor"),
		fctl.WithStringFlag(pollIntervalFlag, "", "Polling interval as a Go duration, for example 30s. Defaults to the service value"),
		fctl.WithStringFlag(configFlag, "", "YAML or JSON configuration file"),
		fctl.WithStringArrayFlag(envFileFlag, nil, "Dotenv configuration file (repeatable)"),
		fctl.WithStringArrayFlag(setFlag, nil, "Configuration value KEY=VALUE, or KEY=@path to read a file (repeatable). Completion follows a --version or --channel typed before it"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*InstallStore](controller),
		fctl.WithFlagUsage("confirm", "Skip the confirmation prompt"),
		fctl.WithExamples(
			"fctl connectivity connections install ee-stripeconnect \\",
			"    --ledger payments --channel beta \\",
			"    --set ENVIRONMENT=sandbox \\",
			"    --set SECRET_KEY=@./stripe.key",
			"fctl connectivity connections install ee-wise \\",
			"    --ledger treasury --env-file wise.env",
		),
		fctl.WithoutFileCompletion(ledgerFlag, nameFlag, pollIntervalFlag, initialCursorFlag),
		fctl.WithRequiredFlag(ledgerFlag),
		fctl.WithFlagCompletion(versionFlag, CompleteVersions(installConnectorArgument)),
		fctl.WithFlagCompletion(channelFlag, connectivityinternal.CompleteChannels()),
		fctl.WithFlagCompletion(setFlag, CompleteSetValues(resolveInstallConnectorVersion, paths)),
	)
}

func validateSelectors(cmd *cobra.Command) error {
	if cmd.Flags().Changed(versionFlag) && cmd.Flags().Changed(channelFlag) {
		return fmt.Errorf("--version and --channel are mutually exclusive; select exactly one")
	}
	for _, flag := range []string{versionFlag, channelFlag} {
		if cmd.Flags().Changed(flag) && strings.TrimSpace(fctl.GetString(cmd, flag)) == "" {
			return fmt.Errorf("--%s cannot be empty", flag)
		}
	}
	if cmd.Flags().Changed(channelFlag) {
		if _, ok := channelPrereleases[fctl.GetString(cmd, channelFlag)]; !ok {
			return fmt.Errorf("unsupported --channel; use stable, rc, beta, or alpha")
		}
	}
	return nil
}

func initialCursor(cmd *cobra.Command) (*string, error) {
	if !cmd.Flags().Changed(initialCursorFlag) {
		return nil, nil
	}
	raw := fctl.GetString(cmd, initialCursorFlag)
	decoded, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil || raw == "" || base64.StdEncoding.EncodeToString(decoded) != raw {
		return nil, fmt.Errorf("--initial-cursor must be non-empty canonical base64")
	}
	return &raw, nil
}

func installConnectorArgument(_ *cobra.Command, args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// resolveInstallConnectorVersion drives --set completion. Without --version
// or --channel it uses the stable release, and falls back to the catalogue's
// latest version when the connector has no stable release yet, so completion
// still lists the fields.
func resolveInstallConnectorVersion(ctx context.Context, client *sdk.FormanceV1, cmd *cobra.Command, args []string) (*connectivity.ConnectorVersion, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, nil
	}
	selector := installVersionSelector(cmd)
	version, err := resolveConnectorVersion(ctx, client, args[0], selector)
	if err == nil || cmd.Flags().Changed(versionFlag) || cmd.Flags().Changed(channelFlag) || !isNotFound(err) {
		// Any other failure keeps the stable selection install would use,
		// so completion never suggests keys from a different schema.
		return version, err
	}
	response, getErr := client.GetConnector(ctx, operations.GetConnectorRequest{Name: url.PathEscape(args[0])}, nil)
	if getErr != nil || response == nil || response.Connector == nil {
		return nil, err
	}
	latest := fctl.StringPointerToString(response.Connector.LatestVersion)
	if latest == "" {
		return nil, err
	}
	return resolveConnectorVersion(ctx, client, args[0], latest)
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

// isNotFound reports a missing resource: the API's not_found error code, or
// a bare 404 the SDK could not decode.
func isNotFound(err error) bool {
	var apiErr *connectivity.Error
	if errors.As(err, &apiErr) {
		return strings.EqualFold(fctl.StringPointerToString(apiErr.ErrorCode), "not_found")
	}
	var sdkErr *sdkerrors.SDKError
	return errors.As(err, &sdkErr) && sdkErr.StatusCode == http.StatusNotFound
}

func (c *InstallController) GetStore() *InstallStore {
	return c.store
}

func (c *InstallController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if err := validateSelectors(cmd); err != nil {
		return nil, err
	}
	if _, err := initialCursor(cmd); err != nil {
		return nil, err
	}
	if strings.TrimSpace(fctl.GetString(cmd, ledgerFlag)) == "" {
		return nil, fmt.Errorf("--ledger cannot be empty")
	}

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	client, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, client.Connectivity.V1)
}

func (c *InstallController) run(cmd *cobra.Command, args []string, client *sdk.FormanceV1) (fctl.Renderable, error) {
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
	config, err := configuration.BuildInstallConfig(cmd, version, configuration.InputOptions{
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
	ledger := fctl.GetString(cmd, ledgerFlag)
	spec := connectivity.ConnectionCreateConnectionConfiguration{
		Connector:     connectorName,
		Ledger:        ledger,
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
		spec.Channel = fctl.Ptr(connectivity.ConnectionCreateChannel(installVersionSelector(cmd)))
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
	response, err := client.CreateConnection(cmd.Context(), connectivity.ConnectionCreate{Name: name, ConnectionConfiguration: spec}, nil)
	var instance *connectivity.Connection
	if response != nil {
		instance = response.Connection
	}
	if err != nil {
		return nil, err
	}
	if instance == nil || instance.Name == "" {
		return nil, fmt.Errorf("install connectivity connection %q: empty response", name)
	}
	c.store.Connection = *instance
	return c, nil
}

func configHasData(config *connectivity.ConnectionConfiguration) bool {
	return config != nil && (len(config.Env) > 0 || len(config.Files) > 0)
}

func (c *InstallController) Render(cmd *cobra.Command, _ []string) error {
	instance := c.store.Connection
	return fctl.RenderText(cmd.OutOrStdout(), fmt.Sprintf("Connection %q installed with connector %q for ledger %q.", instance.Name,
		fctl.StringPointerToString(instance.ConnectionConfiguration.Connector),
		fctl.StringPointerToString(instance.ConnectionConfiguration.Ledger)))
}
