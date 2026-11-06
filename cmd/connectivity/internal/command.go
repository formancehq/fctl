package internal

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

// WithRequiredFlag marks an existing flag as required. Apply it after the flag
// and Args options so a setup error is returned before command execution.
func WithRequiredFlag(name string) fctl.CommandOptionFn {
	return func(cmd *cobra.Command) {
		if err := cmd.MarkFlagRequired(name); err != nil {
			addCommandSetupError(cmd, fmt.Errorf("mark flag --%s required: %w", name, err))
		}
	}
}

// WithFlagCompletion registers completion for an existing flag. Apply it after
// the flag and Args options so a setup error is returned before command execution.
func WithFlagCompletion(name string, completion cobra.CompletionFunc) fctl.CommandOptionFn {
	return func(cmd *cobra.Command) {
		if err := cmd.RegisterFlagCompletionFunc(name, completion); err != nil {
			addCommandSetupError(cmd, fmt.Errorf("register completion for --%s: %w", name, err))
		}
	}
}

func addCommandSetupError(cmd *cobra.Command, setupErr error) {
	validateArgs := cmd.Args
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		err := fmt.Errorf("configure command %q: %w", cmd.CommandPath(), setupErr)
		if validateArgs == nil {
			return err
		}
		return errors.Join(err, validateArgs(cmd, args))
	}
}
