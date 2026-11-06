package clientcontract

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
)

func TestLedgerMixedQueryParamsPreserveTimeBounds(t *testing.T) {
	const input = `{"chart":{},"queries":{"accounts":{"resource":"accounts","params":{"resource":"accounts","pageSize":15,"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z"}}}}`
	var data ledger.V2SchemaDataInput
	require.NoError(t, json.Unmarshal([]byte(input), &data))
	var original map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(input), &original))
	// The server ignores the redundant nested resource; time bounds must survive.
	var actual map[string]struct {
		Resource string         `json:"resource"`
		Params   map[string]any `json:"params"`
	}
	var expected map[string]struct {
		Resource string         `json:"resource"`
		Params   map[string]any `json:"params"`
	}
	require.NoError(t, json.Unmarshal(original["queries"], &expected))
	require.NoError(t, json.Unmarshal([]byte(queryJSON(t, data)), &actual))
	delete(expected["accounts"].Params, "resource")
	delete(actual["accounts"].Params, "resource")
	require.Equal(t, expected, actual)
}

func TestLedgerQueryTemplateRequiresParentResource(t *testing.T) {
	for _, query := range []string{`{"params":{"pageSize":15}}`, `{"resource":"unknown","params":{"pageSize":15}}`, `{"resource":null,"params":{"pageSize":15}}`} {
		var data ledger.V2SchemaDataInput
		err := json.Unmarshal([]byte(`{"chart":{},"queries":{"test":`+query+`}}`), &data)
		require.Error(t, err)
	}
}

func TestLedgerQueryTemplateWithoutParams(t *testing.T) {
	const input = `{"chart":{},"queries":{"logs":{"resource":"logs","vars":{"kind":{"type":"string","default":"CREATED_TRANSACTION"}},"body":{"$match":{"type":"CREATED_TRANSACTION"}}}}}`
	var data ledger.V2SchemaDataInput
	require.NoError(t, json.Unmarshal([]byte(input), &data))
	require.JSONEq(t, `{"logs":{"resource":"logs","vars":{"kind":{"type":"string","default":"CREATED_TRANSACTION"}},"body":{"$match":{"type":"CREATED_TRANSACTION"}}}}`, queryJSON(t, data))
}

func TestLedgerQueryTemplateModelsRemainTyped(t *testing.T) {
	var _ map[string]ledger.V2QueryTemplate = ledger.V2SchemaDataInput{}.V2QueryTemplates
	var _ map[string]ledger.V2QueryTemplate = ledger.V2SchemaData{}.V2QueryTemplates
	_ = ledger.CreateV2QueryTemplateV2AccountQueryTemplate
	_ = ledger.CreateV2QueryTemplateV2TransactionQueryTemplate
	_ = ledger.CreateV2QueryTemplateV2LogQueryTemplate
	_ = ledger.CreateV2QueryTemplateV2VolumeQueryTemplate
}

func TestLedgerRunQueryOverridePreservesVolumeFields(t *testing.T) {
	const input = `{"pageSize":15,"sort":"id:asc","expand":["volumes"],"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z","groupBy":2,"insertionDate":false}`
	var params ledger.V2RunQueryParams
	require.NoError(t, json.Unmarshal([]byte(input), &params))
	actual, err := json.Marshal(params)
	require.NoError(t, err)
	require.JSONEq(t, input, string(actual))
}

func TestLedgerQueryTemplateYAMLPreservesVolumeParams(t *testing.T) {
	const input = `chart: {}
queries:
  volumes:
    resource: volumes
    params:
      pageSize: 15
      expand: [volumes]
      startTime: "2026-01-01T00:00:00Z"
      endTime: "2026-01-02T00:00:00Z"
      groupBy: 2
      insertionDate: false
`
	var document any
	require.NoError(t, yaml.Unmarshal([]byte(input), &document))
	normalized, err := json.Marshal(document)
	require.NoError(t, err)
	var schema ledger.V2SchemaDataInput
	require.NoError(t, json.Unmarshal(normalized, &schema))
	require.JSONEq(t, `{"volumes":{"resource":"volumes","params":{"pageSize":15,"expand":["volumes"],"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z","groupBy":2,"insertionDate":false}}}`, queryJSON(t, schema))
}
