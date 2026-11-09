package configuration

import (
	"fmt"
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
	unconstrained := map[string]any{}

	require.Equal(t, []string{"sandbox", "production"}, enumOf(t, both))
	require.Equal(t, []string{"sandbox", "production"}, enumOf(t, map[string]any{"anyOf": []any{sandbox, production}}), "alternatives combine as a union")
	require.Equal(t, []string{"production"}, enumOf(t, map[string]any{"allOf": []any{both, production}}), "allOf intersects")
	require.Nil(t, enumOf(t, map[string]any{"anyOf": []any{sandbox, unconstrained}}), "an unconstrained alternative accepts any value")
	require.Nil(t, enumOf(t, map[string]any{"oneOf": []any{unconstrained, sandbox}}), "in either order")
	require.Nil(t, enumOf(t, map[string]any{"anyOf": []any{sandbox, true}}), "a boolean alternative is unconstrained")
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

func fieldOf(t *testing.T, section map[string]any, key string) SchemaField {
	t.Helper()
	fields, err := SchemaFields(&connectivity.ConnectorVersion{ConfigSchema: map[string]any{"env": section}})
	require.NoError(t, err)
	return fields[key]
}

func TestSchemaFieldsEnumAlgebra(t *testing.T) {
	tests := map[string]struct {
		field any
		want  []string
	}{
		"oneOf drops values matched by two alternatives": {map[string]any{"oneOf": []any{map[string]any{"enum": []any{"a", "b"}}, map[string]any{"enum": []any{"b", "c"}}}}, []string{"a", "c"}},
		"not makes listed values unknown":                {map[string]any{"enum": []any{"a", "b"}, "not": map[string]any{"enum": []any{"b"}}}, nil},
		"maxLength makes listed values unknown":          {map[string]any{"enum": []any{"a", "bb"}, "maxLength": 1}, nil},
		"const narrows the enum":                         {map[string]any{"enum": []any{"a", "b"}, "const": "a"}, []string{"a"}},
		"const alone is completable":                     {map[string]any{"const": "a"}, []string{"a"}},
		"a mixed enum constrains without completing":     {map[string]any{"allOf": []any{map[string]any{"enum": []any{"a", "b"}}, map[string]any{"enum": []any{"a", 1}}}}, nil},
		"a false alternative adds nothing":               {map[string]any{"anyOf": []any{false, map[string]any{"enum": []any{"a"}}}}, []string{"a"}},
		"an empty intersection adds nothing":             {map[string]any{"anyOf": []any{map[string]any{"allOf": []any{map[string]any{"enum": []any{"a"}}, map[string]any{"enum": []any{"b"}}}}, map[string]any{"enum": []any{"c"}}}}, []string{"c"}},
		"$id starts a new reference resource":            {map[string]any{"$id": "https://example.test/mode", "$defs": map[string]any{"v": map[string]any{"enum": []any{"prod"}}}, "$ref": "#/$defs/v"}, []string{"prod"}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, fieldOf(t, map[string]any{"properties": map[string]any{"MODE": test.field}}, "MODE").Enum)
		})
	}
}

func TestSchemaFieldsWildcardSchemasMakeDeclaredValuesUnknown(t *testing.T) {
	declared := map[string]any{"properties": map[string]any{"MODE": map[string]any{"enum": []any{"a", "b"}}}}

	require.Nil(t, fieldOf(t, map[string]any{"allOf": []any{declared, map[string]any{"additionalProperties": map[string]any{"enum": []any{"b"}}}}}, "MODE").Enum)
	require.Nil(t, fieldOf(t, map[string]any{"properties": declared["properties"], "patternProperties": map[string]any{"^M": map[string]any{"enum": []any{"b"}}}}, "MODE").Enum)
}

func TestSchemaFieldsResolveReferencesThroughArraysAndIgnoreAnchorsInData(t *testing.T) {
	section := map[string]any{
		"allOf":      []any{map[string]any{"properties": map[string]any{"BASE": map[string]any{"enum": []any{"prod"}}}}},
		"default":    map[string]any{"$anchor": "v", "enum": []any{"bogus"}},
		"properties": map[string]any{"ALIAS": map[string]any{"$ref": "#/allOf/0/properties/BASE"}, "ANCHORED": map[string]any{"$ref": "#v"}, "TARGET": map[string]any{"$anchor": "v", "enum": []any{"prod"}}},
	}

	require.Equal(t, []string{"prod"}, fieldOf(t, section, "ALIAS").Enum)
	require.Equal(t, []string{"prod"}, fieldOf(t, section, "ANCHORED").Enum)
}

func TestSchemaFieldsListConditionalFieldsAsOptional(t *testing.T) {
	section := map[string]any{
		"if":               map[string]any{"required": []any{"MODE"}},
		"then":             map[string]any{"required": []any{"TOKEN"}, "properties": map[string]any{"TOKEN": map[string]any{"format": "password"}}},
		"dependentSchemas": map[string]any{"MODE": map[string]any{"properties": map[string]any{"REGION": map[string]any{"type": "string"}}}},
	}

	token := fieldOf(t, section, "TOKEN")
	require.True(t, token.Password)
	require.False(t, token.Required, "a conditional requirement is not global")
	require.Equal(t, "string", fieldOf(t, section, "REGION").Type)
}

func TestSchemaFieldsDisplayMetadata(t *testing.T) {
	section := map[string]any{"properties": map[string]any{
		"SECRET":   map[string]any{"allOf": []any{map[string]any{"type": "string", "writeOnly": true}}},
		"NULLABLE": map[string]any{"type": []any{"string", "null"}},
		"URL":      map[string]any{"type": "string", "allOf": []any{map[string]any{"format": "uri"}}},
	}}

	require.True(t, fieldOf(t, section, "SECRET").Password)
	require.Equal(t, "string | null", fieldOf(t, section, "NULLABLE").Type)
	require.Equal(t, "uri", fieldOf(t, section, "URL").Type)
}

func TestSchemaFieldsWalkSharedReferencesOnce(t *testing.T) {
	defs := map[string]any{"d0": map[string]any{"enum": []any{"a"}}}
	for index := 1; index <= 25; index++ {
		previous := fmt.Sprintf("#/$defs/d%d", index-1)
		defs[fmt.Sprintf("d%d", index)] = map[string]any{"allOf": []any{map[string]any{"$ref": previous}, map[string]any{"$ref": previous}}}
	}

	// 2^25 paths without caching, under the depth limit; this finishes because each
	// definition is walked once.
	field := fieldOf(t, map[string]any{"$defs": defs, "properties": map[string]any{"MODE": map[string]any{"$ref": "#/$defs/d25"}}}, "MODE")

	require.Equal(t, []string{"a"}, field.Enum)
}
