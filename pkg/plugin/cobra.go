package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/TylerBrock/colorjson"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/fctl/pkg/pluginsdk/pluginpb"
)

// BuildCobraCommand translates a plugin's manifest into a cobra command tree.
func BuildCobraCommand(loaded *LoadedPlugin) *cobra.Command {
	manifest, err := loaded.Client.GetManifest(context.Background())
	if err != nil {
		cmd := &cobra.Command{
			Use:   loaded.Name,
			Short: fmt.Sprintf("Plugin %s (failed to load manifest: %v)", loaded.Name, err),
		}
		return cmd
	}

	return buildCommandFromSpec(manifest.RootCommand, loaded)
}

// buildCommandFromSpec recursively converts a CommandSpec into a cobra.Command.
func buildCommandFromSpec(spec *pluginpb.CommandSpec, loaded *LoadedPlugin) *cobra.Command {
	if spec == nil {
		return &cobra.Command{Use: loaded.Name}
	}

	opts := make([]fctl.CommandOption, 0)

	if spec.Short != "" {
		opts = append(opts, fctl.WithShortDescription(spec.Short))
	}
	if spec.Long != "" {
		opts = append(opts, fctl.WithDescription(spec.Long))
	}
	if len(spec.Aliases) > 0 {
		opts = append(opts, fctl.WithAliases(spec.Aliases...))
	}
	if spec.Hidden {
		opts = append(opts, fctl.WithHidden())
	}
	if spec.Deprecated != "" {
		opts = append(opts, fctl.WithDeprecated(spec.Deprecated))
	}
	if spec.Confirm {
		opts = append(opts, fctl.WithConfirmFlag())
	}

	for _, flag := range spec.Flags {
		opts = append(opts, flagSpecToOption(flag, false))
	}
	for _, flag := range spec.PersistentFlags {
		opts = append(opts, flagSpecToOption(flag, true))
	}

	if spec.ArgsConstraint != "" {
		if argsOpt := parseArgsConstraint(spec.ArgsConstraint); argsOpt != nil {
			opts = append(opts, argsOpt)
		}
	}

	if spec.Runnable {
		commandPath := spec.Use
		opts = append(opts, fctl.WithRunE(makePluginRunE(loaded, commandPath, spec.CommandType)))
	}

	var children []*cobra.Command
	for _, sub := range spec.Subcommands {
		children = append(children, buildCommandFromSpec(sub, loaded))
	}
	if len(children) > 0 {
		opts = append(opts, fctl.WithChildCommands(children...))
	}

	switch spec.CommandType {
	case pluginpb.CommandType_COMMAND_TYPE_STACK:
		return fctl.NewStackCommand(spec.Use, opts...)
	case pluginpb.CommandType_COMMAND_TYPE_MEMBERSHIP:
		return fctl.NewMembershipCommand(spec.Use, opts...)
	default:
		return fctl.NewCommand(spec.Use, opts...)
	}
}

// makePluginRunE creates the RunE function that bridges cobra to gRPC plugin execution.
func makePluginRunE(loaded *LoadedPlugin, commandPath string, cmdType pluginpb.CommandType) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		authCtx, err := buildAuthContext(cmd, cmdType)
		if err != nil {
			return err
		}

		flags := collectFlags(cmd)
		outputFormat := fctl.GetString(cmd, fctl.OutputFlag)

		req := &pluginpb.ExecuteRequest{
			CommandPath:  commandPath,
			Args:         args,
			Flags:        flags,
			AuthContext:  authCtx,
			OutputFormat: outputFormat,
		}

		resp, err := loaded.Client.Execute(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("plugin execution failed: %w", err)
		}

		switch r := resp.Result.(type) {
		case *pluginpb.ExecuteResponse_Success:
			return handleSuccess(cmd, r.Success, outputFormat)
		case *pluginpb.ExecuteResponse_Error:
			return fmt.Errorf("plugin error (code %d): %s", r.Error.Code, r.Error.Message)
		default:
			return fmt.Errorf("unexpected response from plugin")
		}
	}
}

// buildAuthContext constructs the AuthContext from the fctl core auth system.
func buildAuthContext(cmd *cobra.Command, cmdType pluginpb.CommandType) (*pluginpb.AuthContext, error) {
	authCtx := &pluginpb.AuthContext{
		InsecureTls: fctl.GetBool(cmd, fctl.InsecureTlsFlag),
		Debug:       fctl.GetBool(cmd, fctl.DebugFlag),
	}

	if cmdType == pluginpb.CommandType_COMMAND_TYPE_BASIC {
		return authCtx, nil
	}

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	authCtx.MembershipUrl = profile.GetMembershipURI()

	if cmdType == pluginpb.CommandType_COMMAND_TYPE_MEMBERSHIP {
		membershipToken, err := fctl.EnsureMembershipAccess(
			cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile,
		)
		if err != nil {
			return nil, err
		}
		authCtx.MembershipToken = membershipToken.Token

		orgID, err := fctl.ResolveOrganizationID(cmd, *profile)
		if err == nil {
			authCtx.OrganizationId = orgID
		}
		return authCtx, nil
	}

	// STACK type
	orgID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackToken, stackAccess, err := fctl.EnsureStackAccess(
		cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, orgID, stackID,
	)
	if err != nil {
		return nil, err
	}

	authCtx.OrganizationId = orgID
	authCtx.StackId = stackID
	authCtx.StackUrl = stackAccess.URI
	authCtx.AccessToken = stackToken.Token

	membershipToken, err := fctl.EnsureMembershipAccess(
		cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile,
	)
	if err == nil {
		authCtx.MembershipToken = membershipToken.Token
	}

	return authCtx, nil
}

// handleSuccess outputs the plugin response to the user.
func handleSuccess(cmd *cobra.Command, success *pluginpb.ExecuteSuccess, outputFormat string) error {
	if outputFormat == "json" && success.JsonData != "" {
		raw := make(map[string]any)
		if err := json.Unmarshal([]byte(success.JsonData), &raw); err == nil {
			f := colorjson.NewFormatter()
			f.Indent = 2
			colorized, err := f.Marshal(raw)
			if err == nil {
				_, err = cmd.OutOrStdout().Write(colorized)
				return err
			}
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), success.JsonData)
		return err
	}

	if success.RenderedText != "" {
		_, err := fmt.Fprint(cmd.OutOrStdout(), success.RenderedText)
		return err
	}
	return nil
}

// collectFlags gathers all resolved flag values from the cobra command.
func collectFlags(cmd *cobra.Command) map[string]string {
	flags := make(map[string]string)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		flags[f.Name] = f.Value.String()
	})
	return flags
}

// flagSpecToOption converts a protobuf FlagSpec to a cobra CommandOption.
func flagSpecToOption(spec *pluginpb.FlagSpec, persistent bool) fctl.CommandOption {
	if persistent {
		switch spec.Type {
		case pluginpb.FlagType_FLAG_TYPE_BOOL:
			defVal := spec.DefaultValue == "true"
			if spec.Shorthand != "" {
				return fctl.WithPersistentBoolPFlag(spec.Name, spec.Shorthand, defVal, spec.Description)
			}
			return fctl.WithPersistentBoolFlag(spec.Name, defVal, spec.Description)
		default:
			if spec.Shorthand != "" {
				return fctl.WithPersistentStringPFlag(spec.Name, spec.Shorthand, spec.DefaultValue, spec.Description)
			}
			return fctl.WithPersistentStringFlag(spec.Name, spec.DefaultValue, spec.Description)
		}
	}

	switch spec.Type {
	case pluginpb.FlagType_FLAG_TYPE_BOOL:
		defVal := spec.DefaultValue == "true"
		return fctl.WithBoolFlag(spec.Name, defVal, spec.Description)
	case pluginpb.FlagType_FLAG_TYPE_INT:
		defVal, _ := strconv.Atoi(spec.DefaultValue)
		return fctl.WithIntFlag(spec.Name, defVal, spec.Description)
	case pluginpb.FlagType_FLAG_TYPE_STRING_SLICE:
		var defVal []string
		if spec.DefaultValue != "" {
			defVal = strings.Split(spec.DefaultValue, ",")
		}
		return fctl.WithStringSliceFlag(spec.Name, defVal, spec.Description)
	default:
		return fctl.WithStringFlag(spec.Name, spec.DefaultValue, spec.Description)
	}
}

// parseArgsConstraint converts an args constraint string to a cobra PositionalArgs option.
func parseArgsConstraint(constraint string) fctl.CommandOption {
	parts := strings.SplitN(constraint, ":", 3)
	switch parts[0] {
	case "exact":
		if len(parts) >= 2 {
			n, _ := strconv.Atoi(parts[1])
			return fctl.WithArgs(cobra.ExactArgs(n))
		}
	case "min":
		if len(parts) >= 2 {
			n, _ := strconv.Atoi(parts[1])
			return fctl.WithArgs(cobra.MinimumNArgs(n))
		}
	case "max":
		if len(parts) >= 2 {
			n, _ := strconv.Atoi(parts[1])
			return fctl.WithArgs(cobra.MaximumNArgs(n))
		}
	case "range":
		if len(parts) >= 3 {
			min, _ := strconv.Atoi(parts[1])
			max, _ := strconv.Atoi(parts[2])
			return fctl.WithArgs(cobra.RangeArgs(min, max))
		}
	case "none":
		return fctl.WithArgs(cobra.NoArgs)
	case "any":
		return fctl.WithArgs(cobra.ArbitraryArgs)
	}
	return nil
}
