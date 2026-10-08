package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/interactive"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

func canSupplyFlag(inputs []pluginsdk.InputSpec, flag pluginsdk.FlagSpec) bool {
	for _, input := range inputs {
		if input.Flag == flag.Name || (flag.Body && input.BodyPointer != "") {
			return true
		}
	}
	return false
}

func relaxedManifest(manifest pluginsdk.Manifest, path []string) pluginsdk.Manifest {
	manifest = cloneManifest(manifest)
	current := &manifest.Root
	for _, name := range path[1:] {
		for i := range current.Subcommands {
			if pluginsdk.CommandName(current.Subcommands[i]) == name {
				current = &current.Subcommands[i]
				break
			}
		}
	}
	current.Args.Min = 0
	current.Confirm = false
	for i := range current.Flags {
		if canSupplyFlag(current.Inputs, current.Flags[i]) {
			current.Flags[i].Required = false
		}
	}
	return manifest
}

func (a *Adapter) runInteractive(cmd *cobra.Command, entry registration, path []string, flags []pluginsdk.FlagSpec, args []string) error {
	spec, err := pluginsdk.FindCommand(entry.manifest, path)
	if err != nil {
		return err
	}
	if spec.Confirm {
		flag := lookupFlag(cmd, "confirm")
		if flag != nil && flag.Changed && flag.Value.String() != "true" {
			return fmt.Errorf("%s requires --confirm", strings.Join(path, " "))
		}
	}
	request, err := prepareRequest(cmd, relaxedManifest(entry.manifest, path), path, flags, args)
	if err != nil {
		return err // Gates and explicit body errors precede auth and discovery.
	}
	request = seedInputContext(spec.Inputs, request)
	instance, choices, err := a.interactiveInstances(cmd, entry, path, &request)
	if err != nil {
		return err
	}
	request, err = collectInputs(cmd, choices, spec.Inputs, request)
	if err != nil {
		return err
	}
	if err := confirmRequest(cmd, spec, path, &request); err != nil {
		return err
	}
	request, err = pluginsdk.NormalizeRequest(entry.manifest, request)
	if err != nil {
		return err
	}
	response, execErr := instance.Execute(cmd.Context(), request)
	if len(response.Data) == 0 {
		return execErr
	}
	return errors.Join(execErr, command.WriteJSON(cmd.OutOrStdout(), response.Data))
}

func (a *Adapter) interactiveInstances(cmd *cobra.Command, entry registration, path []string, request *pluginsdk.ExecuteRequest) (pluginsdk.Plugin, pluginsdk.Plugin, error) {
	service, err := pluginsdk.CommandService(entry.manifest, path)
	if err != nil {
		return nil, nil, err
	}
	client, err := a.resolveClient(cmd.Context(), service, *request)
	if err != nil {
		return nil, nil, err
	}
	if client == nil || client.HTTPClient() == nil {
		return nil, nil, fmt.Errorf("plugin resolver returned no HTTP client")
	}
	request.Endpoint, request.Context = client.Endpoint(), client.Context()
	instance, choices := entry.factory(client.HTTPClient()), entry.factory(choiceClient(client.HTTPClient()))
	if instance == nil || choices == nil {
		return nil, nil, fmt.Errorf("plugin factory returned nil")
	}
	return instance, choices, nil
}

func confirmRequest(cmd *cobra.Command, spec pluginsdk.CommandSpec, path []string, request *pluginsdk.ExecuteRequest) error {
	if !spec.Confirm || request.Flags["confirm"] == "true" {
		return nil
	}
	if request.ChangedFlags["confirm"] {
		return fmt.Errorf("%s requires --confirm", strings.Join(path, " "))
	}
	accepted, err := interactive.Confirm(cmd.Context(), cmd, "Confirm "+confirmationLabel(path, spec.Inputs, *request))
	if err != nil {
		return err
	}
	if !accepted {
		return interactive.ErrCanceled
	}
	request.Flags["confirm"], request.ChangedFlags["confirm"] = "true", true
	return nil
}

func confirmationLabel(path []string, inputs []pluginsdk.InputSpec, request pluginsdk.ExecuteRequest) string {
	parts := append([]string{}, path...)
	values := confirmationValues(inputs, request)
	parts = append(parts, values...)
	return interactive.SafeLabel(strings.Join(parts, " ")) + "?"
}

func confirmationValues(inputs []pluginsdk.InputSpec, request pluginsdk.ExecuteRequest) []string {
	values := append([]string{}, request.Args...)
	values = redactArguments(inputs, values)
	for _, input := range inputs {
		if input.Secret || input.Flag == "data" {
			continue
		}
		value := request.Flags[input.Flag]
		if input.Context != "" {
			value = request.Context[input.Context]
		}
		if value != "" && !slices.Contains(values, value) {
			values = append(values, value)
		}
	}
	return values
}

func collectInputs(cmd *cobra.Command, instance pluginsdk.Plugin, inputs []pluginsdk.InputSpec, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteRequest, error) {
	request.Flags, request.ChangedFlags, request.Context = maps.Clone(request.Flags), maps.Clone(request.ChangedFlags), maps.Clone(request.Context)
	if request.Context == nil {
		request.Context = make(map[string]string)
	}
	request = seedInputContext(inputs, request)
	if !needsInputForm(inputs, request) {
		return request, nil
	}
	for index := 0; index < len(inputs); {
		group, next := inputGroup(inputs, index, request)
		index = next
		if len(group) == 0 {
			continue
		}
		if err := collectGroup(cmd, instance, group, &request); err != nil {
			return request, err
		}
	}
	return request, nil
}

func needsInputForm(inputs []pluginsdk.InputSpec, request pluginsdk.ExecuteRequest) bool {
	required, missing := false, false
	for _, input := range inputs {
		required = required || input.Required
		if !suppliedInput(input, request) {
			if input.Required {
				return true
			}
			missing = true
		}
	}
	return !required && missing
}

func inputGroup(inputs []pluginsdk.InputSpec, index int, request pluginsdk.ExecuteRequest) ([]pluginsdk.InputSpec, int) {
	if suppliedInput(inputs[index], request) {
		return nil, index + 1
	}
	group := []pluginsdk.InputSpec{inputs[index]}
	index++
	// Resolve dependent choice lists after the preceding form is accepted.
	for index < len(inputs) && inputs[index].Source == nil {
		if !suppliedInput(inputs[index], request) {
			group = append(group, inputs[index])
		}
		index++
	}
	return group, index
}

func collectGroup(cmd *cobra.Command, instance pluginsdk.Plugin, inputs []pluginsdk.InputSpec, request *pluginsdk.ExecuteRequest) error {
	fields := make([]interactive.Field, 0, len(inputs))
	for _, input := range inputs {
		field, err := inputField(cmd.Context(), instance, input, *request)
		if err != nil {
			return err
		}
		fields = append(fields, field)
	}
	values, err := interactive.Run(cmd.Context(), cmd, fields)
	if err != nil {
		return err
	}
	for i, input := range inputs {
		if values[i] == "" && !input.Required {
			continue
		}
		if err := bindInput(cmd, input, values[i], request); err != nil {
			return err
		}
	}
	return nil
}

func suppliedInput(input pluginsdk.InputSpec, request pluginsdk.ExecuteRequest) bool {
	if input.AlternativeFlag != "" && request.ChangedFlags[input.AlternativeFlag] {
		return true
	}
	if input.AlternativeArgument != nil && len(request.Args) > *input.AlternativeArgument {
		return true
	}
	switch {
	case input.Flag != "":
		return request.ChangedFlags[input.Flag] || (request.Flags[input.Flag] != "" && input.Source == nil)
	case input.Argument != nil:
		return len(request.Args) > *input.Argument
	case input.Context != "":
		return request.Context[input.Context] != ""
	case input.BodyPointer != "":
		if request.ChangedFlags["data"] {
			return true
		}
		_, exists := bodyValue(request.Body, input.BodyPointer)
		return exists
	}
	return true
}

func inputField(ctx context.Context, instance pluginsdk.Plugin, input pluginsdk.InputSpec, request pluginsdk.ExecuteRequest) (interactive.Field, error) {
	field := interactive.Field{Title: input.Title, Description: input.Description, Kind: input.Kind, Default: input.Default, Required: input.Required, Secret: input.Secret}
	if input.Flag != "" && request.Flags[input.Flag] != "" {
		field.Default = request.Flags[input.Flag]
	}
	for _, option := range input.Options {
		field.Options = append(field.Options, interactive.Option{Label: option.Label, Value: option.Value})
	}
	if input.Source != nil {
		options, err := discoverChoices(ctx, instance, *input.Source, request)
		if err != nil {
			return field, err
		}
		field.Options = options
		if len(options) == 0 {
			message := input.Source.EmptyMessage
			if message == "" {
				message = "no resources available for " + input.Title
			}
			return field, errors.New(message)
		}
	}
	field.Validate = func(value string) error {
		if value == "" && !input.Required {
			return nil
		}
		_, err := inputJSON(input.ValueType, value)
		return err
	}
	return field, nil
}

func inputJSON(kind, value string) (json.RawMessage, error) {
	if kind == "json" || kind == "number" || kind == "bool" {
		if !json.Valid([]byte(value)) {
			return nil, fmt.Errorf("enter valid %s", kind)
		}
		if kind == "number" {
			number, err := decodeJSON(json.RawMessage(value))
			if _, ok := number.(json.Number); err != nil || !ok {
				return nil, fmt.Errorf("enter a JSON number")
			}
		}
		if kind == "bool" && value != "true" && value != "false" {
			return nil, fmt.Errorf("choose true or false")
		}
		return json.RawMessage(value), nil
	}
	return json.Marshal(value)
}

func bindInput(cmd *cobra.Command, input pluginsdk.InputSpec, value string, request *pluginsdk.ExecuteRequest) error {
	switch {
	case input.Flag != "":
		request.Flags[input.Flag], request.ChangedFlags[input.Flag] = value, true
		if input.Flag == "data" {
			request.Body = json.RawMessage(value)
		}
	case input.Argument != nil:
		if len(request.Args) != *input.Argument {
			return fmt.Errorf("interactive arguments must be supplied in order")
		}
		request.Args = append(request.Args, value)
	case input.Context != "":
		request.Context[input.Context] = value
	case input.BodyPointer != "":
		raw, err := inputJSON(input.ValueType, value)
		if err != nil {
			return err
		}
		body, err := setBodyValue(request.Body, input.BodyPointer, raw)
		if err != nil {
			return err
		}
		request.Body = body
	}
	return nil
}

func decodeJSON(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func seedInputContext(inputs []pluginsdk.InputSpec, request pluginsdk.ExecuteRequest) pluginsdk.ExecuteRequest {
	request.Context = maps.Clone(request.Context)
	if request.Context == nil {
		request.Context = make(map[string]string)
	}
	for _, input := range inputs {
		if input.Context != "" && input.AlternativeArgument != nil && len(request.Args) > *input.AlternativeArgument {
			request.Context[input.Context] = request.Args[*input.AlternativeArgument]
		}
	}
	return request
}

func redactArguments(inputs []pluginsdk.InputSpec, values []string) []string {
	for _, input := range inputs {
		if !input.Secret {
			continue
		}
		for _, index := range []*int{input.Argument, input.AlternativeArgument} {
			if index != nil && *index < len(values) {
				values[*index] = "[redacted]"
			}
		}
	}
	return values
}
