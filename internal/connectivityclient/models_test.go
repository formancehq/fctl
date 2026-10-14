package connectivityclient

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectorInstanceSpecHelpers(t *testing.T) {
	value := "configured"
	spec := ConnectorInstanceSpec{
		ConnectorRef: &ObjectRef{Name: "stripe"},
		Env:          map[string]EnvValue{"SETTING": {Value: &value}},
		Files:        []FileMount{{Path: "/etc/plugin/config", Value: &value}},
	}

	require.Equal(t, "stripe", spec.ConnectorName())
	require.Equal(t, &ConnectorInstanceConfig{Env: spec.Env, Files: spec.Files}, spec.Configuration())
	require.Empty(t, (ConnectorInstanceSpec{}).ConnectorName())
	require.Equal(t, &ConnectorInstanceConfig{}, (ConnectorInstanceSpec{}).Configuration())
}

func TestConnectorInstanceSpecOmitsUnsetFields(t *testing.T) {
	for _, cursor := range [][]byte{nil, {}} {
		spec := ConnectorInstanceSpec{Ledger: "main", InitialCursor: cursor}
		body, err := json.Marshal(spec)
		require.NoError(t, err)
		require.JSONEq(t, `{"ledger":"main"}`, string(body))
	}
}

func TestDecodeConnectorInstanceRejectsInvalidInitialCursor(t *testing.T) {
	var instance ConnectorInstance
	err := decodeResponse(strings.NewReader(`{"metadata":{"name":"worker"},"spec":{"ledger":"main","initialCursor":"not-base64!"}}`), &instance, true)
	require.ErrorContains(t, err, "decode connectivity response")
}
