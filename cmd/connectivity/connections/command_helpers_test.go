package connections

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
)

// Tests execute the production command's parsing/rendering and private run method
// with a real SDK over an HTTP fixture, bypassing only interactive authentication.
func testRunCommand[T any](cmd *cobra.Command, controller fctl.Controller[T], run func(*cobra.Command, []string, *sdk.FormanceV1) (fctl.Renderable, error), load testSDKFactory) *cobra.Command {
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup(versionFlag) != nil {
			if err := validateSelectors(cmd); err != nil {
				return err
			}
		}
		if cmd.Name() == "install" && strings.TrimSpace(fctl.GetString(cmd, ledgerFlag)) == "" {
			return fmt.Errorf("--ledger cannot be empty")
		}
		if load == nil {
			return fmt.Errorf("connectivity client factory is required")
		}
		client, err := load(cmd)
		if err != nil {
			return err
		}
		rendered, err := run(cmd, args, client)
		if err != nil {
			return err
		}
		return fctl.WithRender(cmd, args, controller, rendered)
	}
	return cmd
}
func testNewListCommand(load testSDKFactory) *cobra.Command {
	c := NewListController()
	return testRunCommand(NewListCommand(), c, c.run, load)
}
func testNewShowCommand(load testSDKFactory) *cobra.Command {
	c := NewShowController()
	return testRunCommand(NewShowCommand(), c, c.run, load)
}
func testNewInstallCommand(load testSDKFactory, read configuration.ReadFileFunc, paths PathCompleter) *cobra.Command {
	c := NewInstallController(read)
	return testRunCommand(NewInstallCommand(read, paths), c, c.run, load)
}
func testNewConfigureCommand(load testSDKFactory, read configuration.ReadFileFunc, paths PathCompleter) *cobra.Command {
	c := NewConfigureController(read)
	cmd := testRunCommand(NewConfigureCommand(read, paths), c, c.run, load)
	cmd.ValidArgsFunction = testCompleteConnectionNames(load)
	return cmd
}
func testNewSuspendCommand(load testSDKFactory) *cobra.Command {
	c := NewSuspensionController(true)
	cmd := testRunCommand(NewSuspendCommand(), c, c.run, load)
	cmd.ValidArgsFunction = testCompleteConnectionNames(load)
	return cmd
}
func testNewResumeCommand(load testSDKFactory) *cobra.Command {
	c := NewSuspensionController(false)
	cmd := testRunCommand(NewResumeCommand(), c, c.run, load)
	cmd.ValidArgsFunction = testCompleteConnectionNames(load)
	return cmd
}
func testNewUninstallCommand(load testSDKFactory) *cobra.Command {
	c := NewUninstallController()
	cmd := testRunCommand(NewUninstallCommand(), c, c.run, load)
	cmd.ValidArgsFunction = testCompleteConnectionNames(load)
	return cmd
}
func testCompletion(load testSDKFactory, run func(*sdk.FormanceV1, *cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		if load == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		parent := cmd.Context()
		if parent == nil {
			parent = context.Background()
		}
		ctx, cancel := context.WithTimeout(parent, connectivityinternal.CompletionTimeout)
		defer cancel()
		copy := *cmd
		copy.SetContext(ctx)
		client, err := load(&copy)
		if err != nil || client == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return run(client, &copy, args, prefix)
	}
}
func testCompleteConnectionNames(load testSDKFactory) cobra.CompletionFunc {
	return testCompletion(load, func(client *sdk.FormanceV1, cmd *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
		return completeConnectionNamesWithClient(client, cmd, prefix)
	})
}
func testcompleteVersions(load testSDKFactory, resolve ConnectorResolver) cobra.CompletionFunc {
	return testCompletion(load, func(client *sdk.FormanceV1, cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		return completeVersionsWithClient(client, cmd, args, prefix, resolve)
	})
}
func testCompleteVersions(load testSDKFactory, arg func(*cobra.Command, []string) string) cobra.CompletionFunc {
	return testcompleteVersions(load, func(_ context.Context, _ *sdk.FormanceV1, cmd *cobra.Command, args []string) (string, error) {
		if arg == nil {
			return "", nil
		}
		return arg(cmd, args), nil
	})
}
func testCompleteSetValues(load testSDKFactory, resolve ConnectorVersionResolver, paths PathCompleter) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		if strings.Contains(prefix, "=@") || resolve == nil {
			return completeSetValuesWithClient(nil, cmd, args, prefix, resolve, paths)
		}
		return testCompletion(load, func(client *sdk.FormanceV1, cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
			return completeSetValuesWithClient(client, cmd, args, prefix, resolve, paths)
		})(cmd, args, prefix)
	}
}
