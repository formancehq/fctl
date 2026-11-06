package fctl

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCommandSetupReturnsRegistrationErrorsBeforeHooks(t *testing.T) {
	complete := func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	for _, tc := range []struct {
		name    string
		setup   func(*cobra.Command)
		message string
	}{
		{name: "missing required flag", setup: func(cmd *cobra.Command) { WithRequiredFlag("missing")(cmd) }, message: "mark flag --missing required"},
		{name: "missing completion flag", setup: func(cmd *cobra.Command) { WithFlagCompletion("missing", complete)(cmd) }, message: "register completion for --missing"},
		{name: "duplicate completion", setup: func(cmd *cobra.Command) {
			cmd.Flags().String("present", "", "Known flag")
			WithFlagCompletion("present", complete)(cmd)
			WithFlagCompletion("present", complete)(cmd)
		}, message: "already registered"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			persistentPreRunCalled, preRunCalled, runCalled := false, false, false
			root := &cobra.Command{Use: "root", PersistentPreRunE: func(*cobra.Command, []string) error { persistentPreRunCalled = true; return nil }}
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			cmd := &cobra.Command{
				Use: "sample", Args: cobra.ExactArgs(0),
				PreRunE: func(*cobra.Command, []string) error { preRunCalled = true; return nil },
				RunE:    func(*cobra.Command, []string) error { runCalled = true; return nil },
			}
			require.NotPanics(t, func() { tc.setup(cmd) })
			root.AddCommand(cmd)
			root.SetArgs([]string{"sample"})
			err := root.Execute()
			require.ErrorContains(t, err, `configure command "root sample"`)
			require.ErrorContains(t, err, tc.message)
			require.False(t, persistentPreRunCalled)
			require.False(t, preRunCalled)
			require.False(t, runCalled)
		})
	}
}

func TestCommandSetupPreservesAllSetupAndArgumentErrors(t *testing.T) {
	argumentErr := errors.New("argument validation failed")
	argsCalls := 0
	cmd := &cobra.Command{Use: "sample", Args: func(*cobra.Command, []string) error { argsCalls++; return argumentErr }, RunE: func(*cobra.Command, []string) error { t.Fatal("command must not run"); return nil }}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	WithRequiredFlag("ledger")(cmd)
	WithFlagCompletion("name", cobra.NoFileCompletions)(cmd)
	cmd.SetArgs(nil)

	err := cmd.Execute()
	require.ErrorContains(t, err, "mark flag --ledger required")
	require.ErrorContains(t, err, "register completion for --name")
	require.ErrorIs(t, err, argumentErr)
	require.Equal(t, 1, argsCalls)
}
