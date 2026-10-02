package internal

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	FilterFlag = "filter"
	QueryFlag  = "query"
)

func WithListQueryFlags() fctl.CommandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().StringArray(FilterFlag, nil, "Filter expression key=value, key!=value or key~pattern (repeatable)")
		cmd.Flags().String(QueryFlag, "", "Raw filter in the go-libs query dialect (JSON); see /_query/capabilities")
	}
}

// GetListQuery resolves the --filter/--query flags into the query string the
// connectivity API accepts.
func GetListQuery(cmd *cobra.Command) (string, error) {
	filters, err := cmd.Flags().GetStringArray(FilterFlag)
	if err != nil {
		return "", err
	}
	return BuildListQuery(fctl.GetString(cmd, QueryFlag), filters)
}

// BuildListQuery turns --filter expressions into the go-libs query dialect the
// connectivity API accepts on `?query=`, or passes a raw --query through
// untouched. Supported expressions: `key=value` ($match), `key!=value`
// ($exists conjoined with a negated $match, because a bare $not also selects
// objects missing the key) and `key~pattern` ($like, SQL wildcards).
func BuildListQuery(rawQuery string, filters []string) (string, error) {
	if rawQuery != "" {
		if len(filters) > 0 {
			return "", fmt.Errorf("--query cannot be combined with --filter")
		}
		var root any
		if err := json.Unmarshal([]byte(rawQuery), &root); err != nil {
			return "", fmt.Errorf("parse --query: %w", err)
		}
		if _, ok := root.(map[string]any); !ok {
			return "", fmt.Errorf("parse --query: root must be an object")
		}
		return rawQuery, nil
	}
	if len(filters) == 0 {
		return "", nil
	}

	clauses := make([]any, 0, len(filters))
	for _, filter := range filters {
		clause, err := filterClause(filter)
		if err != nil {
			return "", err
		}
		clauses = append(clauses, clause)
	}

	var query any = clauses[0]
	if len(clauses) > 1 {
		query = map[string]any{"$and": clauses}
	}
	encoded, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("encode query: %w", err)
	}
	return string(encoded), nil
}

func filterClause(filter string) (any, error) {
	key, value, operator := splitFilter(filter)
	if operator == "" {
		return nil, fmt.Errorf("parse --filter %q: expected key=value, key!=value or key~pattern", filter)
	}
	if key == "" {
		return nil, fmt.Errorf("parse --filter %q: key is required", filter)
	}
	if value == "" {
		return nil, fmt.Errorf("parse --filter %q: value is required", filter)
	}
	switch operator {
	case "!=":
		return map[string]any{"$and": []any{
			map[string]any{"$exists": map[string]any{key: true}},
			map[string]any{"$not": map[string]any{"$match": map[string]any{key: value}}},
		}}, nil
	case "~":
		return map[string]any{"$like": map[string]any{key: value}}, nil
	default:
		return map[string]any{"$match": map[string]any{key: value}}, nil
	}
}

func splitFilter(filter string) (key, value, operator string) {
	for index := 0; index < len(filter); index++ {
		switch filter[index] {
		case '~', '=':
			return filter[:index], filter[index+1:], string(filter[index])
		case '!':
			if index+1 < len(filter) && filter[index+1] == '=' {
				return filter[:index], filter[index+2:], "!="
			}
		}
	}
	return "", "", ""
}
