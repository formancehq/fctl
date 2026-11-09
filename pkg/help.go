package fctl

import (
	"os"
	"regexp"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// flagUsages is the part of *pflag.FlagSet the usage template needs; naming
// the interface keeps pflag an indirect dependency.
type flagUsages interface {
	FlagUsagesWrapped(cols int) string
}

const requiredFlagMarker = "(required)"

var (
	helpHeadingStyle = pterm.NewStyle(pterm.FgYellow, pterm.Bold)
	helpNameStyle    = pterm.NewStyle(pterm.FgLightCyan)
	helpExampleStyle = pterm.NewStyle(pterm.FgGray)
	helpMarkerStyle  = pterm.NewStyle(pterm.FgYellow)
	helpSections     = regexp.MustCompile(`(?m)^(Aliases|Examples|Available Commands|Additional Commands|Flags|Global Flags|Additional help topics):`)
)

// styledHelp reports whether help may carry colours: only on an interactive
// stdout, and never when NO_COLOR is set.
func styledHelp() bool {
	return TerminalWidth(os.Stdout) > 0 && os.Getenv("NO_COLOR") == ""
}

func helpStyle(style *pterm.Style) func(string) string {
	return func(text string) string {
		if !styledHelp() || text == "" {
			return text
		}
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			if line != "" {
				lines[i] = style.Sprint(line)
			}
		}
		return strings.Join(lines, "\n")
	}
}

func init() {
	cobra.AddTemplateFuncs(map[string]any{
		"fctlHeading":     helpStyle(helpHeadingStyle),
		"fctlCommandName": helpStyle(helpNameStyle),
		"fctlExample":     helpStyle(helpExampleStyle),
		"fctlFlagUsages": func(flags flagUsages) string {
			// Help is written to stdout; zero keeps each description on one
			// line when stdout is not a terminal.
			usages := flags.FlagUsagesWrapped(TerminalWidth(os.Stdout))
			if styledHelp() {
				usages = strings.ReplaceAll(usages, requiredFlagMarker, helpMarkerStyle.Sprint(requiredFlagMarker))
			}
			return usages
		},
	})
}

// WithTerminalHelp fits the help of this command and its subcommands, which
// inherit the usage template, to the terminal: flag descriptions wrap to its
// width, and on an interactive stdout section titles, command names, examples
// and the "(required)" flag marker are coloured.
func WithTerminalHelp() CommandOptionFn {
	return func(cmd *cobra.Command) {
		template := cmd.UsageTemplate()
		template = strings.Replace(template, "Usage:{{", `{{fctlHeading "Usage:"}}{{`, 1)
		template = helpSections.ReplaceAllString(template, `{{fctlHeading "$1:"}}`)
		template = strings.ReplaceAll(template, "{{rpad .Name .NamePadding }}", "{{fctlCommandName (rpad .Name .NamePadding)}}")
		template = strings.ReplaceAll(template, "{{.Example}}", "{{fctlExample .Example}}")
		template = strings.ReplaceAll(template, ".LocalFlags.FlagUsages", "fctlFlagUsages .LocalFlags")
		template = strings.ReplaceAll(template, ".InheritedFlags.FlagUsages", "fctlFlagUsages .InheritedFlags")
		cmd.SetUsageTemplate(template)
	}
}

// WithExamples sets the Examples section of the help, one command per line.
func WithExamples(lines ...string) CommandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Example = "  " + strings.Join(lines, "\n  ")
	}
}

// WithFlagUsage replaces the help text of a flag declared by a shared option.
func WithFlagUsage(name, usage string) CommandOptionFn {
	return func(cmd *cobra.Command) {
		if flag := cmd.Flags().Lookup(name); flag != nil {
			flag.Usage = usage
		}
	}
}

// WithoutFileCompletion stops the shell from offering file names for
// free-text flags. Apply it after the flags are declared.
func WithoutFileCompletion(names ...string) CommandOptionFn {
	return func(cmd *cobra.Command) {
		for _, name := range names {
			if cmd.Flags().Lookup(name) == nil {
				continue
			}
			_ = cmd.RegisterFlagCompletionFunc(name, cobra.NoFileCompletions)
		}
	}
}

// FirstSentence keeps terminal summaries short; -o json carries the full text.
func FirstSentence(text string) string {
	text = strings.TrimSpace(text)
	for i, r := range text {
		if (r == '.' || r == '!' || r == '?') && (i+1 == len(text) || text[i+1] == ' ' || text[i+1] == '\n') {
			return text[:i+1]
		}
	}
	return text
}
