package fctl

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var ErrMissingApproval = errors.New("Missing approval.")

var interactiveContinue = pterm.InteractiveContinuePrinter{
	DefaultValueIndex: 0,
	DefaultText:       "Do you want to continue",
	TextStyle:         &pterm.ThemeDefault.PrimaryStyle,
	Options:           []string{"y", "n"},
	OptionsStyle:      &pterm.ThemeDefault.SuccessMessageStyle,
	SuffixStyle:       &pterm.ThemeDefault.SecondaryStyle,
}

const (
	confirmFlag = "confirm"
)

type approvalPromptKey struct{}

// WithApprovalPrompt replaces the interactive approval prompt for commands using ctx.
// The prompt returns the user's answer; --confirm still bypasses it.
func WithApprovalPrompt(ctx context.Context, prompt func(string) (string, error)) context.Context {
	return context.WithValue(ctx, approvalPromptKey{}, prompt)
}

func showApprovalPrompt(cmd *cobra.Command, disclaimer string) (string, error) {
	text := disclaimer + ".\r\n" + pterm.DefaultInteractiveContinue.DefaultText
	if ctx := cmd.Context(); ctx != nil {
		if prompt, ok := ctx.Value(approvalPromptKey{}).(func(string) (string, error)); ok && prompt != nil {
			return prompt(text)
		}
	}
	return interactiveContinue.WithDefaultText(text).Show()
}

func NeedConfirm(cmd *cobra.Command) bool {
	if GetBool(cmd, confirmFlag) {
		return false
	}
	return true
}

func CheckStackApprobation(cmd *cobra.Command, disclaimer string, args ...any) bool {
	if GetBool(cmd, confirmFlag) {
		return true
	}

	disclaimer = fmt.Sprintf(disclaimer, args...)

	result, err := showApprovalPrompt(cmd, disclaimer)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(result) == "y"
}

func CheckOrganizationApprobation(cmd *cobra.Command, disclaimer string, args ...any) bool {
	if GetBool(cmd, confirmFlag) {
		return true
	}

	result, err := showApprovalPrompt(cmd, disclaimer)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(result) == "y"
}
