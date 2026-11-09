package configuration

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func TestSchemaFieldsBoundsMetadataTraversalDepth(t *testing.T) {
	field := any(map[string]any{"format": "password"})
	for index := 0; index < maxSchemaCollectionDepth+1; index++ {
		field = map[string]any{"allOf": []any{field}}
	}
	version := &connectivity.ConnectorVersion{ConfigSchema: map[string]any{
		"env": map[string]any{"properties": map[string]any{"TOKEN": field}},
	}}

	_, err := SchemaFields(version)

	require.ErrorContains(t, err, "schema nesting exceeds")
}

func enumOf(t *testing.T, field any) []string {
	t.Helper()
	fields, err := SchemaFields(&connectivity.ConnectorVersion{ConfigSchema: map[string]any{
		"env": map[string]any{"properties": map[string]any{"MODE": field}},
	}})
	require.NoError(t, err)
	return fields["MODE"].Enum
}

func TestSchemaFieldsCombineEnumsPerComposition(t *testing.T) {
	sandbox := map[string]any{"enum": []any{"sandbox"}}
	production := map[string]any{"enum": []any{"production"}}
	both := map[string]any{"enum": []any{"sandbox", "production"}}

	require.Equal(t, []string{"sandbox", "production"}, enumOf(t, map[string]any{"enum": []any{"sandbox", "production"}}))
	require.Nil(t, enumOf(t, map[string]any{"anyOf": []any{sandbox, production}}), "alternatives are not merged by keeping the first enum")
	require.Nil(t, enumOf(t, map[string]any{"allOf": []any{both, production}}), "an untracked composition with different enums offers nothing")
	require.Equal(t, []string{"sandbox"}, enumOf(t, map[string]any{"allOf": []any{sandbox, sandbox}}))
	require.Nil(t, enumOf(t, map[string]any{"enum": []any{"a", 1}}), "non-string enums are left to the API")
}

func TestSchemaFieldsCombineEnumsAcrossSectionCompositions(t *testing.T) {
	branch := func(values ...any) map[string]any {
		return map[string]any{"properties": map[string]any{"MODE": map[string]any{"enum": values}}}
	}
	fieldsFor := func(section map[string]any) []string {
		fields, err := SchemaFields(&connectivity.ConnectorVersion{ConfigSchema: map[string]any{"env": section}})
		require.NoError(t, err)
		return fields["MODE"].Enum
	}

	require.Equal(t, []string{"sandbox", "production"}, fieldsFor(map[string]any{"anyOf": []any{branch("sandbox"), branch("production")}}))
	require.Equal(t, []string{"production"}, fieldsFor(map[string]any{"allOf": []any{branch("sandbox", "production"), branch("production")}}))
	require.Nil(t, fieldsFor(map[string]any{"anyOf": []any{branch("sandbox"), map[string]any{}}}), "an open alternative without the field accepts any value")
	require.Nil(t, fieldsFor(map[string]any{"anyOf": []any{map[string]any{}, branch("sandbox")}}), "order of alternatives does not matter")
}

func TestApplyDoubleQuoteEscapesKeepsUnknownEscapesLiteral(t *testing.T) {
	require.Equal(t, `abc\qdef`, applyDoubleQuoteEscapes(`abc\qdef`))
	require.Equal(t, "a\nb\t\"\\", applyDoubleQuoteEscapes(`a\nb\t\"\\`))
}

func TestSchemaFieldsAcceptBooleanSubschemas(t *testing.T) {
	fields, err := SchemaFields(&connectivity.ConnectorVersion{ConfigSchema: map[string]any{
		"env":   map[string]any{"allOf": []any{true, map[string]any{"properties": map[string]any{"TOKEN": map[string]any{"type": "string"}}}}},
		"files": false,
	}})

	require.NoError(t, err)
	require.Contains(t, fields, "TOKEN")
}

func TestBuildInstallConfigRoutesInputForSectionsTheSchemaDoesNotDeclare(t *testing.T) {
	envOnly := &connectivity.ConnectorVersion{ConfigSchema: map[string]any{
		"env": map[string]any{"properties": map[string]any{"TOKEN": map[string]any{"type": "string"}}},
	}}
	noSchema := &connectivity.ConnectorVersion{}

	envConfig, err := BuildInstallConfig(&cobra.Command{}, envOnly, InputOptions{SetValues: []string{"/etc/plugin/config.yaml=content"}}, nil)
	require.NoError(t, err)
	require.Contains(t, envConfig.Files, "/etc/plugin/config.yaml")

	bareConfig, err := BuildInstallConfig(&cobra.Command{}, noSchema, InputOptions{SetValues: []string{"TOKEN=value"}}, nil)
	require.NoError(t, err)
	require.Contains(t, bareConfig.Env, "TOKEN")
}
