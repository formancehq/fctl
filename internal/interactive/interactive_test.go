package interactive_test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/interactive"
)

type testRunner func(context.Context, *cobra.Command, []interactive.Field) ([]string, error)

type enabledCase struct {
	name, ci, noInput    string
	flag, injected, want bool
}

type fieldValidationCase struct {
	name      string
	fields    []interactive.Field
	values    []string
	wantError string
	wantCause error
}

func (r testRunner) Run(ctx context.Context, cmd *cobra.Command, fields []interactive.Field) ([]string, error) {
	return r(ctx, cmd, fields)
}

func bufferedCommand(t *testing.T, runner interactive.Runner) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "fixture"}
	cmd.Flags().Bool("no-input", false, "disable interactive input")
	cmd.SetIn(new(bytes.Buffer))
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	ctx := t.Context()
	if runner != nil {
		ctx = interactive.WithRunner(ctx, runner)
	}
	cmd.SetContext(ctx)
	return cmd
}

func TestEnabledHonorsNoninteractiveControls(t *testing.T) {
	for _, tc := range []enabledCase{
		{name: "buffered runner", injected: true, want: true},
		{name: "buffers without runner"},
		{name: "explicit no-input", flag: true, injected: true},
		{name: "CI", ci: "true", injected: true},
		{name: "numeric CI", ci: "1", injected: true},
		{name: "FCTL_NO_INPUT", noInput: "yes", injected: true},
		{name: "false controls", ci: "FALSE", noInput: "0", injected: true, want: true},
		{name: "zero controls", ci: "0", noInput: "false", injected: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) { assertEnabledCase(t, tc) })
	}
}

func assertEnabledCase(t *testing.T, tc enabledCase) {
	t.Helper()
	t.Setenv("CI", tc.ci)
	t.Setenv("FCTL_NO_INPUT", tc.noInput)
	var runner interactive.Runner
	if tc.injected {
		runner = testRunner(func(context.Context, *cobra.Command, []interactive.Field) ([]string, error) {
			t.Fatal("disabled form or Enabled check invoked the runner")
			return nil, nil
		})
	}
	cmd := bufferedCommand(t, runner)
	if err := cmd.Flags().Set("no-input", map[bool]string{false: "false", true: "true"}[tc.flag]); err != nil {
		t.Fatal(err)
	}
	if got := interactive.Enabled(cmd); got != tc.want {
		t.Fatalf("Enabled = %v, want %v", got, tc.want)
	}
	if !tc.want {
		if _, err := interactive.Run(cmd.Context(), cmd, []interactive.Field{{Title: "Name", Kind: "input"}}); err == nil {
			t.Fatal("Run accepted a form with interactive input disabled")
		}
	}
}

func TestRunUsesInjectedRunnerAndValidatesFields(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	validationErr := errors.New("custom validation failed")
	for _, tc := range []fieldValidationCase{
		{"required", []interactive.Field{{Title: "Name", Kind: "input", Required: true}}, []string{"  "}, "Name is required", nil},
		{"unknown choice", []interactive.Field{{Title: "Ledger", Kind: "select", Options: []interactive.Option{{Label: "Demo", Value: "demo"}}}}, []string{"other"}, "available options", nil},
		{"custom validation", []interactive.Field{{Title: "Amount", Kind: "input", Validate: func(string) error { return validationErr }}}, []string{"invalid"}, validationErr.Error(), validationErr},
		{"too few values", []interactive.Field{{Title: "Name", Kind: "input"}}, nil, "incorrect number of values", nil},
		{"too many values", []interactive.Field{{Title: "Name", Kind: "input"}}, []string{"one", "two"}, "incorrect number of values", nil},
		{"multiple fields", []interactive.Field{{Title: "Name", Kind: "input", Default: "demo", Required: true}, {Title: "Credential", Kind: "input", Description: "Private", Secret: true}}, []string{"demo", "fixture-secret"}, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) { testFieldValidation(t, tc) })
	}
}

func testFieldValidation(t *testing.T, tc fieldValidationCase) {
	t.Helper()
	calls := 0
	runner := testRunner(func(ctx context.Context, cmd *cobra.Command, fields []interactive.Field) ([]string, error) {
		calls++
		if ctx != cmd.Context() {
			t.Fatal("runner did not receive the command context")
		}
		assertFieldMetadata(t, fields, tc.fields)
		return tc.values, nil
	})
	cmd := bufferedCommand(t, runner)
	values, err := interactive.Run(cmd.Context(), cmd, tc.fields)
	if calls != 1 {
		t.Fatalf("runner calls = %d, want 1", calls)
	}
	assertFieldResult(t, values, err, tc)
}

func assertFieldMetadata(t *testing.T, got, want []interactive.Field) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatal("runner did not receive the complete schema")
	}
	for i, field := range got {
		expected := want[i]
		text := [4]string{field.Title, field.Kind, field.Default, field.Description}
		expectedText := [4]string{expected.Title, expected.Kind, expected.Default, expected.Description}
		if text != expectedText || field.Secret != expected.Secret || field.Required != expected.Required || !slices.Equal(field.Options, expected.Options) {
			t.Fatalf("field %d metadata changed", i)
		}
	}
}

func assertFieldResult(t *testing.T, values []string, err error, tc fieldValidationCase) {
	t.Helper()
	if tc.wantError == "" {
		if err != nil || !reflect.DeepEqual(values, tc.values) {
			t.Fatal("runner values were lost or changed")
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), tc.wantError) {
		t.Fatalf("Run error = %v, want %q", err, tc.wantError)
	}
	if tc.wantCause != nil && !errors.Is(err, tc.wantCause) {
		t.Fatal("custom validation error identity was lost")
	}
}

func TestRunCanceledContextDoesNotInvokeRunner(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	cmd := bufferedCommand(t, testRunner(func(context.Context, *cobra.Command, []interactive.Field) ([]string, error) {
		t.Fatal("canceled operation invoked a form")
		return nil, nil
	}))
	ctx, cancel := context.WithCancel(cmd.Context())
	cancel()
	_, err := interactive.Run(ctx, cmd, []interactive.Field{{Title: "Name", Kind: "input"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
}

func TestSelectAndConfirmThroughRunner(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	options := []interactive.Option{{Label: "Demo", Value: "demo"}, {Label: "Production", Value: "prod"}}
	calls := 0
	cmd := bufferedCommand(t, testRunner(func(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
		calls++
		values := make([]string, len(fields))
		for i, field := range fields {
			values[i] = selectorAnswer(t, field, options)
		}
		return values, nil
	}))
	if _, err := interactive.Select(cmd.Context(), cmd, "Empty", nil); err == nil || calls != 0 {
		t.Fatal("an empty selector reached the UI")
	}
	if selected, err := interactive.Select(cmd.Context(), cmd, "Ledger", options); err != nil || selected != "prod" {
		t.Fatalf("Select = %q, %v", selected, err)
	}
	if accepted, err := interactive.Confirm(cmd.Context(), cmd, "Delete prod?"); err != nil || accepted {
		t.Fatalf("Confirm = %v, %v; expected decline", accepted, err)
	}
}

func selectorAnswer(t *testing.T, field interactive.Field, options []interactive.Option) string {
	t.Helper()
	switch field.Kind {
	case "select":
		if field.Title != "Ledger" || !field.Required || !reflect.DeepEqual(field.Options, options) {
			t.Fatal("selector lost its title, choices, or required constraint")
		}
		return "prod"
	case "confirm":
		if field.Title != "Delete prod?" || field.Default != "false" {
			t.Fatal("destructive confirmation lost its title or safe default")
		}
		return "false"
	default:
		t.Fatalf("unexpected field %q of kind %q", field.Title, field.Kind)
		return ""
	}
}

func TestRunPreservesRunnerCancellation(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	cmd := bufferedCommand(t, testRunner(func(context.Context, *cobra.Command, []interactive.Field) ([]string, error) {
		return nil, interactive.ErrCanceled
	}))
	values, err := interactive.Run(cmd.Context(), cmd, []interactive.Field{{Title: "Credential", Kind: "input", Secret: true}})
	if !errors.Is(err, interactive.ErrCanceled) || values != nil {
		t.Fatalf("Run returned %v, %v after cancellation", values, err)
	}
}

func TestRunEscapesDisplayTextWithoutChangingBoundValues(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	const resource = "ledger\x1b[31m\n\u202e"
	fields := []interactive.Field{{
		Title: "Pick\x1b[31m", Description: "Line\nNext\u202e", Kind: "select",
		Options: []interactive.Option{{Label: resource, Value: resource}},
	}}
	cmd := bufferedCommand(t, testRunner(func(_ context.Context, _ *cobra.Command, received []interactive.Field) ([]string, error) {
		field := received[0]
		if field.Title != `Pick\u001b[31m` || field.Description != `Line\u000aNext\u202e` || field.Options[0].Label != `ledger\u001b[31m\u000a\u202e` || field.Options[0].Value != resource {
			t.Fatal("terminal control characters were rendered or the bound identifier changed")
		}
		received[0].Options[0].Label = "runner changed the label"
		return []string{resource}, nil
	}))
	values, err := interactive.Run(cmd.Context(), cmd, fields)
	if err != nil || !reflect.DeepEqual(values, []string{resource}) {
		t.Fatal("escaped display text prevented selection of the original identifier")
	}
	if fields[0].Title != "Pick\x1b[31m" || fields[0].Options[0].Label != resource {
		t.Fatal("rendering mutated the caller's schema")
	}
}
