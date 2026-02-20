package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/apierrors"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/sdkerrors"
	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/logging"

	"github.com/formancehq/fctl/v3/cmd/auth"
	"github.com/formancehq/fctl/v3/cmd/cloud"
	"github.com/formancehq/fctl/v3/cmd/ledger"
	"github.com/formancehq/fctl/v3/cmd/login"
	"github.com/formancehq/fctl/v3/cmd/orchestration"
	"github.com/formancehq/fctl/v3/cmd/payments"
	"github.com/formancehq/fctl/v3/cmd/profiles"
	"github.com/formancehq/fctl/v3/cmd/reconciliation"
	"github.com/formancehq/fctl/v3/cmd/search"
	"github.com/formancehq/fctl/v3/cmd/stack"
	"github.com/formancehq/fctl/v3/cmd/ui"
	"github.com/formancehq/fctl/v3/cmd/version"
	"github.com/formancehq/fctl/v3/cmd/wallets"
	"github.com/formancehq/fctl/v3/cmd/webhooks"
	fctl "github.com/formancehq/fctl/v3/pkg"
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
		),
		fctl.WithPersistentStringPFlag(fctl.ProfileFlag, "p", "", "Configuration profile to use"),
		fctl.WithPersistentStringPFlag(fctl.ConfigDir, "c", fmt.Sprintf("%s/.config/formance/fctl", homedir), "Path to configuration dir"),
		fctl.WithPersistentBoolPFlag(fctl.DebugFlag, "d", false, "Enable debug mode"),
		fctl.WithPersistentStringPFlag(fctl.OutputFlag, "o", "plain", "Output format (plain, json)"),
		fctl.WithPersistentBoolFlag(fctl.InsecureTlsFlag, false, "Allow insecure TLS connections"),
		fctl.WithPersistentBoolFlag(fctl.TelemetryFlag, false, "Enable telemetry"),
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

func Execute() {
	defer func() {
		if e := recover(); e != nil {
			pterm.Error.WithWriter(os.Stderr).Printfln("%s", e)
			debug.PrintStack()
		}
	}()
	ctx, _ := signal.NotifyContext(context.TODO(), os.Interrupt)
	cmd := NewRootCommand()
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
