package cmd

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	stackFlag        = "stack"
	organizationFlag = "organization"
)

func getSelectedOrganization() string {
	return viper.GetString(organizationFlag)
}

func resolveOrganizationID(cmd *cobra.Command, config *internal.Config) (string, error) {
	if id := getSelectedOrganization(); id != "" {
		return id, nil
	}

	client, err := newMembershipClient(cmd, config)
	if err != nil {
		return "", err
	}

	organizations, _, err := client.DefaultApi.ListOrganizations(cmd.Context()).Execute()
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

func getSelectedStack() string {
	return viper.GetString(stackFlag)
}

func resolveStackID(cmd *cobra.Command, config *internal.Config, organizationID string) (string, error) {
	if id := getSelectedStack(); id != "" {
		return id, nil
	}
	client, err := newMembershipClient(cmd, config)
	if err != nil {
		return "", err
	}

	stacks, _, err := client.DefaultApi.ListStacks(cmd.Context(), organizationID).Execute()
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

func withPersistentStringFlag(name, defaultValue, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().String(name, defaultValue, help)
	}
}

func withStringFlag(name, defaultValue, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().String(name, defaultValue, help)
	}
}

func withPersistentStringPFlag(name, short, defaultValue, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().StringP(name, short, defaultValue, help)
	}
}

func withBoolFlag(name string, defaultValue bool, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().Bool(name, defaultValue, help)
	}
}

func withAliases(aliases ...string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Aliases = aliases
	}
}

func withPersistentBoolPFlag(name, short string, defaultValue bool, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().BoolP(name, short, defaultValue, help)
	}
}

func withPersistentBoolFlag(name string, defaultValue bool, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().Bool(name, defaultValue, help)
	}
}

func withIntFlag(name string, defaultValue int, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().Int(name, defaultValue, help)
	}
}

func withStringSliceFlag(name string, defaultValue []string, help string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().StringSlice(name, defaultValue, help)
	}
}

func withHiddenFlag(name string) commandOptionFn {
	return func(cmd *cobra.Command) {
		_ = cmd.Flags().MarkHidden(name)
	}
}

func withRunE(fn func(cmd *cobra.Command, args []string) error) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.RunE = fn
	}
}

func withRun(fn func(cmd *cobra.Command, args []string)) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Run = fn
	}
}

func withChildCommands(cmds ...*cobra.Command) commandOptionFn {
	return func(cmd *cobra.Command) {
		for _, child := range cmds {
			cmd.AddCommand(child)
		}
	}
}

func withShortDescription(v string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Short = v
	}
}

func withArgs(p cobra.PositionalArgs) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Args = p
	}
}

func withDescription(v string) commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Long = v
	}
}

func withSilenceUsage() commandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.SilenceUsage = true
	}
}

func withPersistentPreRunE(fn func(cmd *cobra.Command, args []string) error) commandOptionFn {
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

func newStackCommand(use string, opts ...commandOption) *cobra.Command {
	return newMembershipCommand(use,
		append(opts,
			withPersistentStringFlag(stackFlag, "", "Specific stack (not required if only one stack is present)"),
		)...,
	)
}

func newMembershipCommand(use string, opts ...commandOption) *cobra.Command {
	return newCommand(use,
		append(opts,
			withPersistentStringFlag(organizationFlag, "", "Selected organization (not required if only one organization is present)"),
		)...,
	)
}

func newCommand(use string, opts ...commandOption) *cobra.Command {
	cmd := &cobra.Command{
		Use: use,
	}
	for _, opt := range opts {
		opt.apply(cmd)
	}
	return cmd
}
