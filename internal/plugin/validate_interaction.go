package plugin

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func validateInteraction(manifest pluginsdk.Manifest) error {
	return validateCommandInteraction(manifest, manifest.Root, nil)
}

func validateCommandInteraction(manifest pluginsdk.Manifest, command pluginsdk.CommandSpec, inherited map[string]pluginsdk.FlagSpec) error {
	if !slices.Contains([]string{"", "identity", "organization", "stack"}, command.Target) {
		return fmt.Errorf("command %q: invalid target %q", command.Use, command.Target)
	}
	if len(command.Inputs) > 0 && (!command.Runnable || len(command.Subcommands) > 0) {
		return fmt.Errorf("command %q: inputs require a runnable leaf", command.Use)
	}
	flags := interactionFlags(command, inherited, false)
	if err := validateInteractionInputs(manifest, command, flags); err != nil {
		return err
	}
	next := interactionFlags(command, inherited, true)
	for _, child := range command.Subcommands {
		if err := validateCommandInteraction(manifest, child, next); err != nil {
			return err
		}
	}
	return nil
}

func validateInteractionInputs(manifest pluginsdk.Manifest, command pluginsdk.CommandSpec, flags map[string]pluginsdk.FlagSpec) error {
	bindings := make(map[string]bool)
	for _, input := range command.Inputs {
		if err := validateInteractionInput(manifest, command, flags, input); err != nil {
			return fmt.Errorf("command %q input %q: %w", command.Use, input.Title, err)
		}
		for _, binding := range interactionBindings(input) {
			if bindings[binding] {
				return fmt.Errorf("command %q: duplicate input binding %q", command.Use, binding)
			}
			bindings[binding] = true
		}
	}
	return nil
}

func interactionFlags(command pluginsdk.CommandSpec, inherited map[string]pluginsdk.FlagSpec, persistentOnly bool) map[string]pluginsdk.FlagSpec {
	flags := maps.Clone(inherited)
	if flags == nil {
		flags = make(map[string]pluginsdk.FlagSpec)
	}
	for _, flag := range command.Flags {
		if !persistentOnly || flag.Persistent {
			flags[flag.Name] = flag
		}
	}
	return flags
}

func validateInteractionInput(manifest pluginsdk.Manifest, command pluginsdk.CommandSpec, flags map[string]pluginsdk.FlagSpec, input pluginsdk.InputSpec) error {
	checks := []func() error{
		func() error { return validateInputPresentation(input) },
		func() error { return validateInputBinding(command, flags, input) },
		func() error { return validateInputAlternatives(command, flags, input) },
		func() error { return validateInputChoices(input, flags) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	if input.Default != "" {
		if err := validateInputValue(input, flags, input.Default); err != nil {
			return fmt.Errorf("invalid default: %w", err)
		}
	}
	if input.Source != nil {
		return validateChoiceSource(manifest, command, flags, *input.Source)
	}
	return nil
}

func validateInputPresentation(input pluginsdk.InputSpec) error {
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("input title is required")
	}
	if !slices.Contains([]string{"input", "text", "select", "confirm"}, input.Kind) {
		return fmt.Errorf("invalid input kind %q", input.Kind)
	}
	if !slices.Contains([]string{"", "string", "json", "bool", "number"}, input.ValueType) {
		return fmt.Errorf("invalid input value type %q", input.ValueType)
	}
	if input.Kind == "confirm" && input.ValueType != "" && input.ValueType != "bool" {
		return fmt.Errorf("confirm input requires a boolean value type")
	}
	if input.Secret && (input.Kind != "input" || (input.ValueType != "" && input.ValueType != "string") || input.Default != "") {
		return fmt.Errorf("secret input requires a password field without a default")
	}
	return nil
}

func validateInputBinding(command pluginsdk.CommandSpec, flags map[string]pluginsdk.FlagSpec, input pluginsdk.InputSpec) error {
	bindings := interactionBindings(input)
	if len(bindings) != 1 {
		return fmt.Errorf("input requires exactly one binding")
	}
	if input.Flag != "" {
		flag, exists := flags[input.Flag]
		if !exists {
			return fmt.Errorf("input refers to undeclared flag %q", input.Flag)
		}
		if input.Secret && flag.Type != "string" {
			return fmt.Errorf("secret input requires a string flag")
		}
	}
	if input.Argument != nil {
		return validateInputArgument(command, *input.Argument)
	}
	if input.Context != "" && !slices.Contains([]string{"organization", "stack"}, input.Context) {
		return fmt.Errorf("invalid input context %q", input.Context)
	}
	if input.Secret && input.Context != "" {
		return fmt.Errorf("resource context cannot be a secret input")
	}
	if input.BodyPointer != "" {
		if !slices.ContainsFunc(slices.Collect(maps.Values(flags)), func(flag pluginsdk.FlagSpec) bool { return flag.Body }) {
			return fmt.Errorf("body input requires a declared body flag")
		}
		return validateInputPointer(input.BodyPointer)
	}
	return nil
}

func validateInputArgument(command pluginsdk.CommandSpec, index int) error {
	if index < 0 || index >= command.Args.Max {
		return fmt.Errorf("input argument index %d is outside command bounds", index)
	}
	return nil
}

func validateInputAlternatives(command pluginsdk.CommandSpec, flags map[string]pluginsdk.FlagSpec, input pluginsdk.InputSpec) error {
	if input.AlternativeArgument != nil {
		if err := validateInputArgument(command, *input.AlternativeArgument); err != nil {
			return fmt.Errorf("alternative argument: %w", err)
		}
	}
	if input.AlternativeFlag != "" {
		if _, exists := flags[input.AlternativeFlag]; !exists {
			return fmt.Errorf("alternative refers to undeclared flag %q", input.AlternativeFlag)
		}
	}
	return nil
}

func interactionBindings(input pluginsdk.InputSpec) []string {
	var bindings []string
	for name, value := range map[string]string{"flag": input.Flag, "body": input.BodyPointer, "context": input.Context} {
		if value != "" {
			bindings = append(bindings, name+":"+value)
		}
	}
	if input.Argument != nil {
		bindings = append(bindings, fmt.Sprintf("argument:%d", *input.Argument))
	}
	return bindings
}

func validateInputPointer(pointer string) error {
	if !utf8.ValidString(pointer) || !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("body input requires an RFC 6901 JSON pointer")
	}
	for index := 0; index < len(pointer); index++ {
		if pointer[index] == '~' {
			index++
			if index == len(pointer) || (pointer[index] != '0' && pointer[index] != '1') {
				return fmt.Errorf("invalid JSON pointer escape")
			}
		}
	}
	return nil
}

func validateInputChoices(input pluginsdk.InputSpec, flags map[string]pluginsdk.FlagSpec) error {
	if len(input.Options) > 0 && input.Source != nil {
		return fmt.Errorf("input options and source are mutually exclusive")
	}
	if input.Kind != "select" {
		if len(input.Options) > 0 || input.Source != nil {
			return fmt.Errorf("only select inputs can declare choices")
		}
		return nil
	}
	if len(input.Options) == 0 && input.Source == nil {
		return fmt.Errorf("select input requires options or a source")
	}
	seen := make(map[string]bool)
	for _, option := range input.Options {
		if strings.TrimSpace(option.Label) == "" || (input.Required && option.Value == "") || seen[option.Value] {
			return fmt.Errorf("input options require labels and distinct usable values")
		}
		seen[option.Value] = true
		if err := validateInputValue(input, flags, option.Value); err != nil {
			return fmt.Errorf("invalid option value: %w", err)
		}
	}
	return nil
}

func validateInputValue(input pluginsdk.InputSpec, flags map[string]pluginsdk.FlagSpec, value string) error {
	kind := cmp.Or(input.ValueType, "string")
	if input.Kind == "confirm" {
		kind = "bool"
	}
	if err := validateInteractionValue(kind, value); err != nil {
		return err
	}
	if flag, exists := flags[input.Flag]; exists {
		flag.Default = value
		return validateDefault(flag)
	}
	return nil
}

func validateInteractionValue(kind, value string) error {
	switch kind {
	case "string":
		return nil
	case "bool":
		if value == "true" || value == "false" {
			return nil
		}
	case "json", "number":
		if !json.Valid([]byte(value)) {
			break
		}
		if kind == "json" {
			return nil
		}
		decoder := json.NewDecoder(bytes.NewReader([]byte(value)))
		decoder.UseNumber()
		var decoded any
		if err := decoder.Decode(&decoded); err == nil {
			if _, ok := decoded.(json.Number); ok {
				return nil
			}
		}
	}
	return fmt.Errorf("value must be a valid %s", kind)
}

func validateChoiceSource(manifest pluginsdk.Manifest, owner pluginsdk.CommandSpec, ownerFlags map[string]pluginsdk.FlagSpec, source pluginsdk.ChoiceSource) error {
	command, flags, err := findChoiceSourceCommand(manifest, source.CommandPath)
	if err != nil {
		return err
	}
	if err := validateChoiceSourceHeader(command, flags, source); err != nil {
		return err
	}
	for _, argument := range source.Args {
		if err := validateChoiceReference(owner, ownerFlags, argument); err != nil {
			return err
		}
	}
	return validateChoiceSourceFlags(owner, ownerFlags, flags, source.Flags)
}

func validateChoiceSourceHeader(command pluginsdk.CommandSpec, flags map[string]pluginsdk.FlagSpec, source pluginsdk.ChoiceSource) error {
	if len(source.Args) < command.Args.Min || len(source.Args) > command.Args.Max {
		return fmt.Errorf("choice source arguments are outside command bounds")
	}
	if strings.TrimSpace(source.ValueField) == "" {
		return fmt.Errorf("choice source value field is required")
	}
	for _, label := range source.LabelFields {
		if strings.TrimSpace(label) == "" {
			return fmt.Errorf("choice source label fields must not be empty")
		}
	}
	if err := validateChoiceExclusions(source.ExcludeTrueFields); err != nil {
		return err
	}
	if err := validateChoiceMatches(source.MatchFields); err != nil {
		return err
	}
	return validateChoicePagination(flags, source)
}

var choiceFieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func validateChoiceExclusions(fields []string) error {
	seen := make(map[string]bool)
	for _, field := range fields {
		if !choiceFieldName.MatchString(field) {
			return fmt.Errorf("choice source excluded field %q must be a valid field name", field)
		}
		if seen[field] {
			return fmt.Errorf("choice source has duplicate excluded field %q", field)
		}
		seen[field] = true
	}
	return nil
}

func validateChoiceMatches(fields map[string][]string) error {
	for field, values := range fields {
		if !choiceFieldName.MatchString(field) {
			return fmt.Errorf("choice source match field %q must be a valid field name", field)
		}
		if len(values) == 0 {
			return fmt.Errorf("choice source match field %q requires at least one value", field)
		}
		seen := make(map[string]bool)
		for _, value := range values {
			if value == "" {
				return fmt.Errorf("choice source match field %q requires nonempty values", field)
			}
			if seen[value] {
				return fmt.Errorf("choice source match field %q has duplicate value %q", field, value)
			}
			seen[value] = true
		}
	}
	return nil
}

func validateChoicePagination(flags map[string]pluginsdk.FlagSpec, source pluginsdk.ChoiceSource) error {
	if source.AfterField == "" {
		return nil
	}
	if !choiceFieldName.MatchString(source.AfterField) {
		return fmt.Errorf("choice source after field must be a valid field name")
	}
	if flag, exists := flags["after"]; !exists || flag.Type != "string" {
		return fmt.Errorf("choice source after field requires a declared string after flag")
	}
	if flag, exists := flags["page-size"]; !exists || flag.Type != "uint32" {
		return fmt.Errorf("choice source after field requires a declared uint32 page-size flag")
	}
	return nil
}

func findChoiceSourceCommand(manifest pluginsdk.Manifest, path []string) (pluginsdk.CommandSpec, map[string]pluginsdk.FlagSpec, error) {
	command, err := pluginsdk.FindCommand(manifest, path)
	if err != nil || !command.Runnable {
		return pluginsdk.CommandSpec{}, nil, fmt.Errorf("choice source must name a runnable command in the same plugin")
	}
	var flags map[string]pluginsdk.FlagSpec
	for index := range path {
		current, err := pluginsdk.FindCommand(manifest, path[:index+1])
		if err != nil {
			return pluginsdk.CommandSpec{}, nil, err
		}
		flags = interactionFlags(current, flags, index < len(path)-1)
	}
	return command, flags, nil
}

func validateChoiceSourceFlags(owner pluginsdk.CommandSpec, ownerFlags, sourceFlags map[string]pluginsdk.FlagSpec, values map[string]string) error {
	for name, value := range values {
		flag, exists := sourceFlags[name]
		if !exists {
			return fmt.Errorf("choice source refers to undeclared flag %q", name)
		}
		if strings.HasPrefix(value, "$") {
			if err := validateChoiceReference(owner, ownerFlags, value); err != nil {
				return err
			}
		} else {
			flag.Default = value
			if err := validateDefault(flag); err != nil {
				return fmt.Errorf("choice source flag: %w", err)
			}
		}
	}
	return nil
}

func validateChoiceReference(owner pluginsdk.CommandSpec, flags map[string]pluginsdk.FlagSpec, value string) error {
	name, reference := strings.CutPrefix(value, "$")
	if !reference {
		return nil
	}
	if raw, argument := strings.CutPrefix(name, "arg"); argument {
		if index, err := strconv.Atoi(raw); err == nil {
			return validateInputArgument(owner, index)
		}
	}
	if _, exists := flags[name]; exists || name == "organization" || name == "stack" {
		return nil
	}
	if slices.ContainsFunc(owner.Inputs, func(input pluginsdk.InputSpec) bool { return input.Context == name && name != "" }) {
		return nil
	}
	return fmt.Errorf("choice source refers to undeclared input %q", name)
}
