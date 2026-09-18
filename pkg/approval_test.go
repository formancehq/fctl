package fctl

import (
	"context"
	"errors"
	"testing"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestApprobation(t *testing.T) {
	checks := map[string]func(*cobra.Command, string, ...any) bool{
		"stack":        CheckStackApprobation,
		"organization": CheckOrganizationApprobation,
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name     string
				args     []string
				answer   string
				approved bool
				prompts  int
			}{
				{name: "accept", answer: "y", approved: true, prompts: 1},
				{name: "accept uppercase", answer: "Y", approved: true, prompts: 1},
				{name: "refuse", answer: "n", prompts: 1},
				{name: "refuse uppercase", answer: "N", prompts: 1},
				{name: "empty answer", prompts: 1},
				{name: "unexpected answer", answer: "yes", prompts: 1},
				{name: "confirm bypasses prompt", args: []string{"--confirm"}, approved: true},
				{name: "explicit false still prompts", args: []string{"--confirm=false"}, answer: "n", prompts: 1},
			} {
				t.Run(tc.name, func(t *testing.T) {
					cmd := NewCommand("action", WithConfirmFlag())
					require.NoError(t, cmd.ParseFlags(tc.args))
					prompts := 0
					cmd.SetContext(WithApprovalPrompt(context.Background(), func(text string) (string, error) {
						prompts++
						require.Equal(t, "Disable stack.\r\n"+pterm.DefaultInteractiveContinue.DefaultText, text)
						return tc.answer, nil
					}))

					require.Equal(t, tc.prompts != 0, NeedConfirm(cmd))
					require.Equal(t, tc.approved, check(cmd, "Disable stack"))
					require.Equal(t, tc.prompts, prompts)
				})
			}
		})
	}
}

func TestStackApprobationFormatsDisclaimer(t *testing.T) {
	cmd := NewCommand("action", WithConfirmFlag())
	cmd.SetContext(WithApprovalPrompt(context.Background(), func(text string) (string, error) {
		require.Equal(t, "Disable stack 'sandbox'.\r\n"+pterm.DefaultInteractiveContinue.DefaultText, text)
		return "n", nil
	}))
	require.False(t, CheckStackApprobation(cmd, "Disable stack '%s'", "sandbox"))
}

func TestApprobationPromptError(t *testing.T) {
	for name, check := range map[string]func(*cobra.Command, string, ...any) bool{
		"stack":        CheckStackApprobation,
		"organization": CheckOrganizationApprobation,
	} {
		t.Run(name, func(t *testing.T) {
			cmd := NewCommand("action", WithConfirmFlag())
			errPrompt := errors.New("terminal unavailable")
			cmd.SetContext(WithApprovalPrompt(context.Background(), func(string) (string, error) {
				return "y", errPrompt
			}))
			require.PanicsWithValue(t, errPrompt, func() { check(cmd, "Disable stack") })
		})
	}
}
