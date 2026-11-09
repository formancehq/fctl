package fctl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestFirstSentenceStopsAtTheFirstSentenceEnd(t *testing.T) {
	require.Equal(t, "Bound Stripe environment: sandbox or live.", FirstSentence("Bound Stripe environment: sandbox or live. Every object is checked."))
	require.Equal(t, "Version v1.2 only.", FirstSentence("Version v1.2 only."))
	require.Equal(t, "No period", FirstSentence("  No period  "))
}

func TestWithoutFileCompletionReplacesTheShellFileFallback(t *testing.T) {
	cmd := NewCommand("install", WithStringFlag("ledger", "", ""), WithoutFileCompletion("ledger", "missing"))

	completion, ok := cmd.GetFlagCompletionFunc("ledger")

	require.True(t, ok)
	_, directive := completion(cmd, nil, "")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	_, ok = cmd.GetFlagCompletionFunc("missing")
	require.False(t, ok)
}

func TestWithTerminalHelpGivenNarrowTerminalThenFlagDescriptionsWrapAndExamplesShow(t *testing.T) {
	restore := SetTerminalWidth(50)
	defer restore()
	root := NewCommand("connectivity", WithTerminalHelp())
	child := NewCommand("install",
		WithStringFlag("ledger", "", "(required) Target ledger. Immutable once set, even after the connection is suspended"),
		WithBoolFlag("confirm", false, "Confirm action"),
		WithFlagUsage("confirm", "Skip the confirmation prompt"),
		WithExamples("fctl connectivity connections install ee-wise --ledger treasury"),
	)
	child.Run = func(*cobra.Command, []string) {}
	root.AddCommand(child)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"install", "--help"})

	require.NoError(t, root.Execute())

	help := out.String()
	require.Contains(t, help, "Examples:\n  fctl connectivity connections install ee-wise --ledger treasury")
	require.Contains(t, help, "Skip the\n")
	require.NotContains(t, help, "Confirm action")
	for _, line := range strings.Split(help, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			require.LessOrEqual(t, len(line), 50, "flag line %q", line)
		}
	}
}

func TestWithTerminalHelpStylesSectionsCommandsExamplesAndFlags(t *testing.T) {
	cmd := NewCommand("connectivity", WithTerminalHelp())

	template := cmd.UsageTemplate()

	for _, marker := range []string{`{{fctlHeading "Usage:"}}`, `{{fctlHeading "Flags:"}}`, `{{fctlHeading "Global Flags:"}}`, "fctlCommandName (rpad .Name .NamePadding)", "fctlExample .Example", "fctlFlagUsages .LocalFlags", "fctlFlagUsages .InheritedFlags"} {
		require.Contains(t, template, marker)
	}
}

func TestTerminalHelpGivenNoColorThenHelpHasNoEscapeCodes(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	restore := SetTerminalWidth(80)
	defer restore()
	root := NewCommand("connectivity", WithTerminalHelp(), WithExamples("fctl connectivity server-infos"))
	child := NewCommand("install", WithStringFlag("ledger", "", "(required) Target ledger"))
	child.Run = func(*cobra.Command, []string) {}
	root.AddCommand(child)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})

	require.NoError(t, root.Execute())

	require.NotContains(t, out.String(), "\x1b[")
	require.Contains(t, out.String(), "\n  install")
}

func TestHelpStyleLeavesTextUnchangedWithoutATerminal(t *testing.T) {
	restore := SetTerminalWidth(0)
	defer restore()

	require.Equal(t, "Usage:", helpStyle(helpHeadingStyle)("Usage:"))
}
