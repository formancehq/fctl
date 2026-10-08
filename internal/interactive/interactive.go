// Package interactive owns terminal forms. Plugins describe fields using the
// public SDK; they never depend on this package or on the terminal UI library.
package interactive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

var ErrCanceled = errors.New("operation canceled")

type Option struct{ Label, Value string }

type Field struct {
	Title, Description, Kind, Default string
	Required, Secret                  bool
	Options                           []Option
	Validate                          func(string) error
}

// Runner is an injectable terminal boundary for command integration tests.
type Runner interface {
	Run(context.Context, *cobra.Command, []Field) ([]string, error)
}

type runnerKey struct{}

func WithRunner(ctx context.Context, runner Runner) context.Context {
	return context.WithValue(ctx, runnerKey{}, runner)
}

func Enabled(cmd *cobra.Command) bool {
	disabled := flagValue(cmd, "no-input") == "true"
	if disabled || truthy(os.Getenv("FCTL_NO_INPUT")) || truthy(os.Getenv("CI")) {
		return false
	}
	if runner, ok := cmd.Context().Value(runnerKey{}).(Runner); ok && runner != nil {
		return true
	}
	input, inOK := cmd.InOrStdin().(*os.File)
	output, outOK := cmd.ErrOrStderr().(*os.File)
	return inOK && outOK && term.IsTerminal(input.Fd()) && term.IsTerminal(output.Fd())
}

func truthy(value string) bool {
	return value != "" && value != "0" && !strings.EqualFold(value, "false")
}

func Select(ctx context.Context, cmd *cobra.Command, title string, options []Option) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no choices available for %s", title)
	}
	values, err := Run(ctx, cmd, []Field{{Title: title, Kind: "select", Options: options, Required: true}})
	if err != nil {
		return "", err
	}
	return values[0], nil
}

func Confirm(ctx context.Context, cmd *cobra.Command, title string) (bool, error) {
	values, err := Run(ctx, cmd, []Field{{Title: title, Kind: "confirm", Default: "false"}})
	if err != nil {
		return false, err
	}
	return values[0] == "true", nil
}

func Run(ctx context.Context, cmd *cobra.Command, fields []Field) ([]string, error) {
	fields = safeFields(fields)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !Enabled(cmd) {
		return nil, fmt.Errorf("interactive input requires a terminal; provide arguments and flags")
	}
	if runner, ok := ctx.Value(runnerKey{}).(Runner); ok && runner != nil {
		values, err := runner.Run(ctx, cmd, fields)
		if err != nil {
			return nil, err
		}
		if len(values) != len(fields) {
			return nil, fmt.Errorf("interactive runner returned an incorrect number of values")
		}
		for i, field := range fields {
			if err := validate(field, values[i]); err != nil {
				return nil, err
			}
		}
		return values, nil
	}
	return runTerminal(ctx, cmd, fields)
}

// SafeLabel prevents resource names from becoming terminal control sequences.
// Bound values stay exact; only text displayed by the terminal is escaped.
func SafeLabel(value string) string {
	var result []string
	for _, r := range value {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			result = append(result, fmt.Sprintf("\\u%04x", r))
		} else {
			result = append(result, string(r))
		}
	}
	return strings.Join(result, "")
}

func safeFields(fields []Field) []Field {
	result := make([]Field, len(fields))
	for i, field := range fields {
		field.Title, field.Description = SafeLabel(field.Title), SafeLabel(field.Description)
		field.Options = append([]Option{}, field.Options...)
		for j := range field.Options {
			field.Options[j].Label = SafeLabel(field.Options[j].Label)
		}
		result[i] = field
	}
	return result
}

func validate(field Field, value string) error {
	if field.Required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field.Title)
	}
	if field.Kind == "select" {
		found := false
		for _, option := range field.Options {
			found = found || option.Value == value
		}
		if !found {
			return fmt.Errorf("choose one of the available options")
		}
	}
	if field.Validate != nil {
		return field.Validate(value)
	}
	return nil
}

func runTerminal(ctx context.Context, cmd *cobra.Command, fields []Field) ([]string, error) {
	values := make([]string, len(fields))
	booleans := make([]bool, len(fields))
	form := terminalForm(cmd, fields, values, booleans)
	color := flagValue(cmd, "color")
	if color == "never" || (color != "always" && os.Getenv("NO_COLOR") != "") {
		form.WithProgramOptions(tea.WithColorProfile(colorprofile.Ascii))
	} else if color == "always" {
		form.WithProgramOptions(tea.WithColorProfile(colorprofile.TrueColor))
	}
	// WithProgramOptions replaces Huh's program options. Bind streams last so
	// forced color profiles cannot redirect terminal controls into JSON stdout.
	form.WithInput(cmd.InOrStdin()).WithOutput(cmd.ErrOrStderr())
	if err := form.RunWithContext(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(err, huh.ErrUserAborted) || errors.Is(err, tea.ErrProgramKilled) {
			return nil, ErrCanceled
		}
		return nil, err
	}
	for i, field := range fields {
		if field.Kind == "confirm" {
			values[i] = strconv.FormatBool(booleans[i])
		}
	}
	return values, nil
}

func flagValue(cmd *cobra.Command, name string) string {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag.Value.String()
	}
	return ""
}

func terminalControl(field Field, value *string, boolean *bool) huh.Field {
	validator := func(value string) error { return validate(field, value) }
	switch field.Kind {
	case "select":
		options := make([]huh.Option[string], 0, len(field.Options))
		for _, option := range field.Options {
			options = append(options, huh.NewOption(option.Label, option.Value))
		}
		return huh.NewSelect[string]().Title(field.Title).Description(field.Description).Options(options...).Value(value).Validate(validator)
	case "confirm":
		return huh.NewConfirm().Title(field.Title).Description(field.Description).Value(boolean)
	case "text":
		return huh.NewText().Title(field.Title).Description(field.Description).Value(value).Validate(validator)
	default:
		input := huh.NewInput().Title(field.Title).Description(field.Description).Value(value).Validate(validator)
		if field.Secret {
			input.EchoMode(huh.EchoModePassword)
		}
		return input
	}
}

func terminalForm(cmd *cobra.Command, fields []Field, values []string, booleans []bool) *huh.Form {
	groups := make([]*huh.Group, 0, len(fields))
	for i, field := range fields {
		values[i] = field.Default
		booleans[i] = field.Default == "true"
		group := huh.NewGroup(terminalControl(field, &values[i], &booleans[i])).Title(cmd.CommandPath())
		if len(fields) > 1 {
			group.Description(fmt.Sprintf("Step %d of %d · Ctrl+C to cancel", i+1, len(fields)))
		}
		if len(fields) == 1 {
			group.Description("Ctrl+C to cancel")
		}
		groups = append(groups, group)
	}
	return huh.NewForm(groups...)
}
