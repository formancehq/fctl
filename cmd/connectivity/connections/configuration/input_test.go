package configuration

import (
	"testing"

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
