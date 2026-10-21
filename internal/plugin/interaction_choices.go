package plugin

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/interactive"
)

func discoverChoices(ctx context.Context, instance pluginsdk.Plugin, source pluginsdk.ChoiceSource, request pluginsdk.ExecuteRequest) ([]interactive.Option, error) {
	query, err := choiceQuery(source, request)
	if err != nil {
		return nil, err
	}
	options, err := enumerateChoices(ctx, instance, source, query)
	if err != nil {
		return nil, err
	}
	if source.PreferredPrefix != "" {
		slices.SortStableFunc(options, func(a, b interactive.Option) int {
			ap, bp := strings.HasPrefix(a.Value, source.PreferredPrefix), strings.HasPrefix(b.Value, source.PreferredPrefix)
			if ap == bp {
				return 0
			}
			if ap {
				return -1
			}
			return 1
		})
	}
	return options, nil
}

func choiceQuery(source pluginsdk.ChoiceSource, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteRequest, error) {
	query := pluginsdk.ExecuteRequest{CommandPath: source.CommandPath, Endpoint: request.Endpoint, Context: maps.Clone(request.Context), Flags: map[string]string{}, ChangedFlags: map[string]bool{}}
	for _, arg := range source.Args {
		value, err := choiceReference(arg, request)
		if err != nil {
			return query, err
		}
		query.Args = append(query.Args, value)
	}
	for flag, reference := range source.Flags {
		value, err := choiceReference(reference, request)
		if err != nil {
			return query, err
		}
		query.Flags[flag], query.ChangedFlags[flag] = value, true
	}
	if source.AfterField != "" {
		query.Flags["page-size"], query.ChangedFlags["page-size"] = "100", true
	}
	return query, nil
}

func enumerateChoices(ctx context.Context, instance pluginsdk.Plugin, source pluginsdk.ChoiceSource, query pluginsdk.ExecuteRequest) ([]interactive.Option, error) {
	var options []interactive.Option
	seen, cursors := map[string]bool{}, map[string]bool{}
	for range 1000 {
		items, more, next, err := readChoicePage(ctx, instance, query, source)
		if err != nil {
			return nil, err
		}
		options, err = appendChoices(options, items, source, seen)
		if err != nil {
			return nil, err
		}
		if !more {
			return options, nil
		}
		if next == "" || cursors[next] {
			return nil, fmt.Errorf("cannot enumerate choices; specify the resource explicitly")
		}
		cursors[next] = true
		flag := "cursor"
		if source.AfterField != "" {
			flag = "after"
		}
		query.Flags[flag], query.ChangedFlags[flag] = next, true
	}
	return nil, fmt.Errorf("choice pagination exceeded its limit; specify the resource explicitly")
}

func readChoicePage(ctx context.Context, instance pluginsdk.Plugin, query pluginsdk.ExecuteRequest, source pluginsdk.ChoiceSource) ([]any, bool, string, error) {
	response, err := instance.Execute(ctx, query)
	if err != nil {
		return nil, false, "", fmt.Errorf("list choices: %w", err)
	}
	value, err := decodeJSON(response.Data)
	if err != nil {
		return nil, false, "", fmt.Errorf("read choices: %w", err)
	}
	items, more, next, err := choiceItems(value)
	if err != nil {
		return nil, false, "", err
	}
	if source.AfterField != "" {
		return keysetChoicePage(items, source.AfterField)
	}
	return items, more, next, nil
}

func appendChoices(options []interactive.Option, items []any, source pluginsdk.ChoiceSource, seen map[string]bool) ([]interactive.Option, error) {
	for _, item := range items {
		if excludedChoice(item, source.ExcludeTrueFields) || !matchingChoice(item, source.MatchFields) {
			continue
		}
		option, err := choiceOption(item, source)
		if err != nil {
			return nil, err
		}
		if !seen[option.Value] {
			options = append(options, option)
			seen[option.Value] = true
		}
	}
	if len(options) > 10000 {
		return nil, fmt.Errorf("too many choices; specify the resource explicitly")
	}
	return options, nil
}

func excludedChoice(item any, fields []string) bool {
	object, ok := item.(map[string]any)
	if !ok {
		return false
	}
	for _, field := range fields {
		if value, ok := choiceField(object, field).(bool); ok && value {
			return true
		}
	}
	return false
}

func matchingChoice(item any, fields map[string][]string) bool {
	object, ok := item.(map[string]any)
	if !ok {
		return len(fields) == 0
	}
	for field, values := range fields {
		if !slices.Contains(values, choiceText(choiceField(object, field))) {
			return false
		}
	}
	return true
}

func choiceReference(reference string, request pluginsdk.ExecuteRequest) (string, error) {
	if !strings.HasPrefix(reference, "$") {
		return reference, nil
	}
	name := reference[1:]
	if suffix, ok := strings.CutPrefix(name, "arg"); ok {
		index, err := strconv.Atoi(suffix)
		if err == nil && index >= 0 && index < len(request.Args) {
			return request.Args[index], nil
		}
	}
	if value := request.Flags[name]; value != "" {
		return value, nil
	}
	if value := request.Context[name]; value != "" {
		return value, nil
	}
	return "", fmt.Errorf("select %s before listing dependent choices", name)
}

func choiceItems(value any) ([]any, bool, string, error) {
	if items, ok := value.([]any); ok {
		return items, false, "", nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, false, "", fmt.Errorf("resource list is not an object or array")
	}
	if cursor, ok := object["cursor"].(map[string]any); ok {
		object = cursor
	}
	items, ok := object["data"].([]any)
	if !ok {
		return nil, false, "", fmt.Errorf("resource list has no data array")
	}
	more := false
	if value, exists := object["hasMore"]; exists {
		parsed, ok := value.(bool)
		if !ok {
			return nil, false, "", fmt.Errorf("resource list has invalid hasMore value")
		}
		more = parsed
	}
	return items, more, choiceText(object["next"]), nil
}

func choiceOption(item any, source pluginsdk.ChoiceSource) (interactive.Option, error) {
	object, ok := item.(map[string]any)
	if !ok {
		return interactive.Option{}, fmt.Errorf("resource choice is not an object")
	}
	value := choiceText(choiceField(object, source.ValueField))
	if value == "" {
		return interactive.Option{}, fmt.Errorf("resource choice lacks %s", source.ValueField)
	}
	var labels []string
	for _, field := range source.LabelFields {
		if label := choiceText(choiceField(object, field)); label != "" && label != value {
			labels = append(labels, label)
		}
	}
	label := value
	if len(labels) > 0 {
		label = strings.Join(labels, " · ") + " (" + value + ")"
	}
	return interactive.Option{Label: label, Value: value}, nil
}

func choiceField(object map[string]any, field string) any {
	if value, exists := object[field]; exists {
		return value
	}
	var value any = object
	for part := range strings.SplitSeq(field, ".") {
		current, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = current[part]
	}
	return value
}

func choiceText(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

func keysetChoicePage(items []any, field string) ([]any, bool, string, error) {
	if len(items) < 100 {
		return items, false, "", nil
	}
	last, ok := items[len(items)-1].(map[string]any)
	if !ok {
		return nil, false, "", fmt.Errorf("resource choice is not an object")
	}
	value := choiceText(choiceField(last, field))
	if value == "" {
		return nil, false, "", fmt.Errorf("resource choice lacks pagination field %s", field)
	}
	return items, true, value, nil
}
