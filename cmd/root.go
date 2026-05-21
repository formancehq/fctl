package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/apierrors"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/sdkerrors"
	"github.com/formancehq/go-libs/v4/api"
	"github.com/formancehq/go-libs/v4/logging"

	"github.com/formancehq/fctl/v3/cmd/auth"
	"github.com/formancehq/fctl/v3/cmd/cloud"
	"github.com/formancehq/fctl/v3/cmd/ledger"
	"github.com/formancehq/fctl/v3/cmd/login"
	"github.com/formancehq/fctl/v3/cmd/orchestration"
	"github.com/formancehq/fctl/v3/cmd/payments"
	plugincmd "github.com/formancehq/fctl/v3/cmd/plugin"
	"github.com/formancehq/fctl/v3/cmd/profiles"
	"github.com/formancehq/fctl/v3/cmd/reconciliation"
	"github.com/formancehq/fctl/v3/cmd/search"
	"github.com/formancehq/fctl/v3/cmd/stack"
	"github.com/formancehq/fctl/v3/cmd/ui"
	"github.com/formancehq/fctl/v3/cmd/version"
	"github.com/formancehq/fctl/v3/cmd/wallets"
	"github.com/formancehq/fctl/v3/cmd/webhooks"
	fctl "github.com/formancehq/fctl/v3/pkg"
	pluginpkg "github.com/formancehq/fctl/v3/pkg/plugin"
)

func init() {
	cobra.EnableTraverseRunHooks = true
}

func NewRootCommand() *cobra.Command {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	cmd := fctl.NewCommand("fctl",
		fctl.WithSilenceError(),
		fctl.WithShortDescription("Formance Control CLI"),
		fctl.WithChildCommands(
			ui.NewCommand(),
			version.NewCommand(),
			login.NewCommand(),
			NewPromptCommand(),
			ledger.NewCommand(),
			payments.NewCommand(),
			reconciliation.NewCommand(),
			profiles.NewCommand(),
			stack.NewCommand(),
			auth.NewCommand(),
			cloud.NewCommand(),
			search.NewCommand(),
			webhooks.NewCommand(),
			wallets.NewCommand(),
			orchestration.NewCommand(),
			plugincmd.NewCommand(),
		),
		fctl.WithPersistentStringPFlag(fctl.ProfileFlag, "p", "", "Configuration profile to use"),
		fctl.WithPersistentStringPFlag(fctl.ConfigDir, "c", fmt.Sprintf("%s/.config/formance/fctl", homedir), "Path to configuration dir"),
		fctl.WithPersistentBoolPFlag(fctl.DebugFlag, "d", false, "Enable debug mode"),
		fctl.WithPersistentStringPFlag(fctl.OutputFlag, "o", "plain", "Output format (plain, json)"),
		fctl.WithPersistentBoolFlag(fctl.InsecureTlsFlag, false, "Allow insecure TLS connections"),
		fctl.WithPersistentBoolFlag(fctl.TelemetryFlag, false, "Enable telemetry"),
		fctl.WithPersistentStringFlag("plugin-binary", "", "Load a plugin for one-shot use (name=/path/to/binary)"),
		fctl.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			logger := logging.NewDefaultLogger(cmd.OutOrStdout(), fctl.GetBool(cmd, fctl.DebugFlag), false, false)
			ctx := logging.ContextWithLogger(cmd.Context(), logger)
			cmd.SetContext(ctx)
			return nil
		}),
	)

	cmd.Version = version.Version
	err = cmd.RegisterFlagCompletionFunc(fctl.ProfileFlag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		profiles, err := fctl.ListProfiles(cmd)
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveError
		}

		return profiles, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		panic(err)
	}
	return cmd
}

// parsePluginBinaryArg scans os.Args for --plugin-binary=name=/path or
// --plugin-binary name=/path and returns the value, or empty string if absent.
func parsePluginBinaryArg() string {
	for i, arg := range os.Args {
		if arg == "--plugin-binary" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, "--plugin-binary=") {
			return strings.TrimPrefix(arg, "--plugin-binary=")
		}
	}
	return ""
}

func removeChildCommand(parent *cobra.Command, name string) {
	for _, child := range parent.Commands() {
		if child.Name() == name {
			parent.RemoveCommand(child)
			return
		}
	}
}

// serviceCommands maps command names to their service name for plugin resolution.
// The value indicates whether the built-in command covers the service (true = v2 built-in exists).
var serviceCommands = map[string]struct {
	serviceName    string
	builtInCovers  func(version string) bool
}{
	"ledger": {
		serviceName: "ledger",
		builtInCovers: func(version string) bool {
			// Built-in ledger commands support v1/v2 (< 3.0.0)
			v, err := semver.NewVersion(version)
			if err != nil {
				return true // can't parse, fall back to built-in
			}
			return v.Major() < 3
		},
	},
}

// wrapWithPluginResolution wraps a service command's PersistentPreRunE to
// detect the service version at runtime and resolve between plugin and built-in.
func wrapWithPluginResolution(
	cmd *cobra.Command,
	serviceName string,
	builtInCovers func(string) bool,
	pm *pluginpkg.PluginManager,
	registry *pluginpkg.RegistryClient,
) {
	originalPreRunE := cmd.PersistentPreRunE

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Run the original PersistentPreRunE first (logger setup, etc.)
		if originalPreRunE != nil {
			if err := originalPreRunE(cmd, args); err != nil {
				return err
			}
		}

		// Try to detect service version — this requires a stack client.
		// If we can't get one (no profile, no auth), fall through to built-in.
		_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
		if err != nil {
			return nil // can't auth, let built-in handle (it will show its own error)
		}

		stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
		if err != nil {
			return nil // can't get stack client, let built-in handle
		}

		serviceVersion, err := pluginpkg.DetectServiceVersion(cmd.Context(), stackClient, serviceName)
		if err != nil {
			return nil // can't detect version, let built-in handle
		}

		resolution, err := pluginpkg.Resolve(serviceName, serviceVersion, pm, registry, builtInCovers(serviceVersion))
		if err != nil {
			return nil
		}

		switch r := resolution.(type) {
		case pluginpkg.UsePlugin:
			return pluginpkg.ReplaceCommandTree(cmd, r.Plugin)
		case pluginpkg.UseBuiltIn:
			return nil
		case pluginpkg.NeedInstall:
			loaded, err := pluginpkg.AutoDiscover(cmd.Context(), r, pm, registry)
			if err != nil {
				return err
			}
			return pluginpkg.ReplaceCommandTree(cmd, loaded)
		}

		return nil
	}
}

func Execute() {
	defer func() {
		if e := recover(); e != nil {
			pterm.Error.WithWriter(os.Stderr).Printfln("%s", e)
			debug.PrintStack()
		}
	}()
	ctx, _ := signal.NotifyContext(context.TODO(), os.Interrupt)
	cmd := NewRootCommand()

	// Initialize plugin infrastructure
	configDir := fmt.Sprintf("%s/.config/formance/fctl", os.Getenv("HOME"))
	if dir := os.Getenv("FCTL_CONFIG_DIR"); dir != "" {
		configDir = dir
	}

	debug := fctl.GetBool(cmd, fctl.DebugFlag)
	pm := pluginpkg.NewPluginManager(configDir, debug)
	pm.DiscoverAndLoad(ctx)
	defer pm.Shutdown()

	// Handle --plugin-binary for ephemeral one-shot plugin loading.
	// Parsed from os.Args because cobra flags aren't parsed yet at this point.
	if pluginBinary := parsePluginBinaryArg(); pluginBinary != "" {
		if name, binaryPath, ok := strings.Cut(pluginBinary, "="); ok && name != "" && binaryPath != "" {
			var opts []pluginpkg.LoadPluginOption
			if debug {
				opts = append(opts, pluginpkg.WithDebug())
			}
			loaded, err := pluginpkg.LoadPlugin(name, binaryPath, opts...)
			if err != nil {
				pterm.Error.WithWriter(os.Stderr).Printfln("Failed to load plugin %s: %v", name, err)
			} else {
				loaded.Version = "ephemeral"
				loaded.CompatibleWith = ">= 0.0.0"
				pluginCmd := pluginpkg.BuildCobraCommand(loaded)
				removeChildCommand(cmd, name)
				cmd.AddCommand(pluginCmd)
				defer loaded.Kill()
			}
		}
	}

	registry := pluginpkg.NewRegistryClient(fctl.GetHttpClient(cmd))

	// Wrap service commands with plugin resolution
	for _, child := range cmd.Commands() {
		if svc, ok := serviceCommands[child.Name()]; ok {
			wrapWithPluginResolution(child, svc.serviceName, svc.builtInCovers, pm, registry)
		}
	}

	if err := cmd.ExecuteContext(ctx); err != nil {
		switch {
		case errors.Is(err, fctl.ErrMissingApproval):
			pterm.Error.WithWriter(os.Stderr).Printfln("Command aborted as you didn't approve.")
			os.Exit(1)
		case fctl.IsInvalidAuthentication(err):
			pterm.Error.WithWriter(os.Stderr).Printfln("Your authentication is invalid, please login :)")
		default:
			unwrapped := err
			for unwrapped != nil {
				//notes(gfyrag): not a clean assertion but following errors does not implements standard Is() helper for errors
				switch err := unwrapped.(type) {
				case *sdkerrors.ErrorResponse:
					printErrorResponse(err)
					return
				case *sdkerrors.V2ErrorResponse:
					printV2ErrorResponse(err)
					return
				case *apierrors.APIError:
					body := err.Body

					errResponse := api.ErrorResponse{}
					if err := json.Unmarshal([]byte(body), &errResponse); err != nil {
						pterm.Error.WithWriter(os.Stderr).Printf("%s\r\n", body)
						return
					}
					printError(errResponse.ErrorCode, errResponse.ErrorMessage, &errResponse.Details)
					return
				case *apierrors.Error:
					errMsg := ""
					if err.ErrorMessage != nil {
						errMsg = *err.ErrorMessage
					}
					printError(err.ErrorCode, errMsg, nil)
					return
				default:
					pterm.Error.WithWriter(os.Stderr).Println(unwrapped)
					unwrapped = errors.Unwrap(unwrapped)
				}
			}
		}

		os.Exit(255)
	}
}

func printError(code string, message string, details *string) {
	pterm.Error.WithWriter(os.Stderr).Printfln("Got error with code %s: %s", code, message)
	if details != nil && *details != "" {
		pterm.Error.WithWriter(os.Stderr).Printfln("Details:\r\n%s", *details)
	}
	os.Exit(2)
}

func printV2ErrorResponse(target *sdkerrors.V2ErrorResponse) {
	printError(string(target.ErrorCode), target.ErrorMessage, target.Details)
}

func printErrorResponse(target *sdkerrors.ErrorResponse) {
	printError(string(target.ErrorCode), target.ErrorMessage, target.Details)
}
