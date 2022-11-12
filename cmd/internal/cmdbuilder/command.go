package cmdbuilder

import (
	"context"

	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	stackFlag        = "stack"
	organizationFlag = "organization"
)

var (
	ErrOrganizationNotSpecified = errors.New("organization not specified")
)

func GetSelectedOrganization() string {
	return viper.GetString(organizationFlag)
}

func RetrieveOrganizationIDFromFlagOrProfile(cfg *config.Config) (string, error) {
	if id := GetSelectedOrganization(); id != "" {
		return id, nil
	}

	if defaultOrganization := config.GetCurrentProfile(cfg).GetDefaultOrganization(); defaultOrganization != "" {
		return defaultOrganization, nil
	}
	return "", ErrOrganizationNotSpecified
}

func ResolveOrganizationID(ctx context.Context, cfg *config.Config) (string, error) {
	if id, err := RetrieveOrganizationIDFromFlagOrProfile(cfg); err == nil {
		return id, nil
	}

	client, err := membership.NewClient(ctx, cfg)
	if err != nil {
		return "", err
	}

	organizations, _, err := client.DefaultApi.ListOrganizations(ctx).Execute()
	if err != nil {
		return "", errors.Wrap(err, "listing organizations")
	}

	if len(organizations.Data) == 0 {
		return "", errors.New("no organizations found")
	}

	if len(organizations.Data) > 1 {
		return "", errors.New("found more than one organization and no organization specified")
	}

	return organizations.Data[0].Id, nil
}

func GetSelectedStack() string {
	return viper.GetString(stackFlag)
}

func ResolveStackID(ctx context.Context, cfg *config.Config, organizationID string) (string, error) {
	if id := GetSelectedStack(); id != "" {
		return id, nil
	}
	client, err := membership.NewClient(ctx, cfg)
	if err != nil {
		return "", err
	}

	stacks, _, err := client.DefaultApi.ListStacks(ctx, organizationID).Execute()
	if err != nil {
		return "", errors.Wrap(err, "listing stacks")
	}
	if len(stacks.Data) == 0 {
		return "", errors.New("no stacks found")
	}
	if len(stacks.Data) > 1 {
		return "", errors.New("found more than one stack and no stack specified")
	}
	return stacks.Data[0].Id, nil
}

type commandOption interface {
	apply(cmd *cobra.Command)
}
type commandOptionFn func(cmd *cobra.Command)

func (fn commandOptionFn) apply(cmd *cobra.Command) {
	fn(cmd)
}

func WithPersistentStringFlag(name, defaultValue, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().String(name, defaultValue, help)
	}
}

func WithStringFlag(name, defaultValue, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().String(name, defaultValue, help)
	}
}

func WithPersistentStringPFlag(name, short, defaultValue, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().StringP(name, short, defaultValue, help)
	}
}

func WithBoolFlag(name string, defaultValue bool, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().Bool(name, defaultValue, help)
	}
}

func WithAliases(aliases ...string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Aliases = aliases
	}
}

func WithPersistentBoolPFlag(name, short string, defaultValue bool, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().BoolP(name, short, defaultValue, help)
	}
}

func WithPersistentBoolFlag(name string, defaultValue bool, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().Bool(name, defaultValue, help)
	}
}

func WithIntFlag(name string, defaultValue int, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().Int(name, defaultValue, help)
	}
}

func WithStringSliceFlag(name string, defaultValue []string, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().StringSlice(name, defaultValue, help)
	}
}

func WithHiddenFlag(name string) commandOptionFn {
	return func(cmd *cobra.Command) {
		_ = cmd.Flags().MarkHidden(name)
	}
}

func WithRunE(fn func(cmd *cobra.Command, args []string) error) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.RunE = fn
	}
}

func WithRun(fn func(cmd *cobra.Command, args []string)) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Run = fn
	}
}

func WithChildCommands(cmds ...*cobra.Command) commandOptionFn {
	return func(cmd *cobra.Command) {
		for _, child := range cmds {
			cmd.AddCommand(child)
		}
	}
}

func WithShortDescription(v string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Short = v
	}
}

func WithArgs(p cobra.PositionalArgs) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Args = p
	}
}

func WithValidArgs(validArgs ...string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.ValidArgs = validArgs
	}
}

func WithValidArgsFunction(fn func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.ValidArgsFunction = fn
	}
}

func WithDescription(v string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Long = v
	}
}

func WithSilenceUsage() commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.SilenceUsage = true
	}
}

func WithSilenceError() commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.SilenceErrors = true
	}
}

func WithPersistentPreRunE(fn func(cmd *cobra.Command, args []string) error) commandOptionFn {
	return func(cmd *cobra.Command) {
		oldPersistentPreRunE := cmd.PersistentPreRunE
		cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			if oldPersistentPreRunE != nil {
				if err := oldPersistentPreRunE(cmd, args); err != nil {
					return err
				}
			}
			if err := fn(cmd, args); err != nil {
				return err
			}
			originalCommand := cmd
			ctx := cmd.Context()
			for {
				cmd = cmd.Parent()
				if cmd == nil {
					return nil
				}
				if cmd.PersistentPreRunE != nil {
					cmd.SetContext(ctx)
					if err := cmd.PersistentPreRunE(cmd, args); err != nil {
						return err
					}
					originalCommand.SetContext(cmd.Context())
				}
			}
		}
	}
}

func NewStackCommand(use string, opts ...commandOption) *cobra.Command {
	return NewMembershipCommand(use,
		append(opts,
			WithPersistentStringFlag(stackFlag, "", "Specific stack (not required if only one stack is present)"),
		)...,
	)
}

func NewMembershipCommand(use string, opts ...commandOption) *cobra.Command {
	return NewCommand(use,
		append(opts,
			WithPersistentStringFlag(organizationFlag, "", "Selected organization (not required if only one organization is present)"),
		)...,
	)
}

func NewCommand(use string, opts ...commandOption) *cobra.Command {
	cmd := &cobra.Command{
		Use: use,
	}
	for _, opt := range opts {
		opt.apply(cmd)
	}
	return cmd
}
