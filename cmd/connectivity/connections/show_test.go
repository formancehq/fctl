package connections

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type showInstanceClientMock struct {
	get func(context.Context, string) (*connectivity.Connection, error)
}

func TestShowRendersPortableStatusAndConfigurationValues(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.ConnectionConfiguration.Env = map[string]string{"TOKEN": "must-not-leak"}
	instance.ConnectionConfiguration.Files = map[string]string{"/etc/plugin/config.yaml": "also-must-not-leak"}
	output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), "stripe-eu")
	require.NoError(t, err)
	for _, value := range []string{"stripe-eu", "stripe", "main", "Desired Configuration", "Applied Version", "2.0.0", "active", "running", "42", "Last Error (may be historical)", "source temporarily unavailable", "TOKEN", "/etc/plugin/config.yaml", "literal"} {
		require.Contains(t, output, value)
	}
	require.Contains(t, output, "must-not-leak")
	require.Contains(t, output, "also-must-not-leak")
	for _, value := range []string{"Namespace", "UID", "Resource Version", "Conditions", "Resolved Image", "Source Tip Sequence"} {
		require.NotContains(t, output, value)
	}
}

func TestShowPlainOutputShowsReturnedConfigValues(t *testing.T) {
	instance := instanceWithTwoFiles()
	instance.ConnectionConfiguration.Env = map[string]string{
		"API_KEY": "must-not-leak",
	}
	command := testNewShowCommand(factoryWithInstance(*instance))

	output, err := executeCommand(command, "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "API_KEY")
	require.Contains(t, output, "literal")
	require.Contains(t, output, "must-not-leak")
	require.Contains(t, output, "private config")
}

func TestShowPlainOutputOmitsUnsupportedStartSequence(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.ConnectionConfiguration.InitialCursor = stringPtr("b3BhcXVlLWluaXRpYWwtY3Vyc29y")

	output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.NotContains(t, output, "Start Sequence")
	require.NotContains(t, output, "987654321")
}

func TestShowConnectionJSONPreservesCompleteModel(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	setInstanceConfig(&instance, &connectivity.ConnectionConfiguration{Env: map[string]string{
		"API_KEY": "json-keeps-full-model",
	}})
	command := testNewShowCommand(factoryWithInstance(instance))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "--output", "json", "stripe-eu")

	require.NoError(t, err)
	var envelope struct {
		Data ShowStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, instance, envelope.Data.Connection)
}

func TestShowConnectionHandlesMissingStatusAndConfig(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.ConnectionStatus = nil
	setInstanceConfig(&instance, nil)

	output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "No configuration entries.")
}

func TestShowConnectionRendersSuspensionAndOrderedProvenance(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.ConnectionConfiguration.Suspend = fctl.Ptr(true)
	instance.ConnectionStatus.SuspendedBy = []connectivity.SuspensionSource{
		{Source: "request", Name: "stripe-eu"},
		{Source: "policy", Name: "maintenance"},
	}

	output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "Suspend")
	require.Contains(t, output, "true")
	require.NotContains(t, output, "Replicas")
	require.Contains(t, output, "Suspended by")
	require.Contains(t, output, "request/stripe-eu, policy/maintenance")
}

func TestShowConnectionReturnsAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (*sdk.FormanceV1, error)
		want    string
	}{
		"API": {
			factory: factoryReturning(showInstanceClientMock{get: func(context.Context, string) (*connectivity.Connection, error) {
				return nil, errors.New("connection unavailable")
			}}),
			want: "connection unavailable",
		},
		"empty response": {
			factory: factoryReturning(showInstanceClientMock{get: func(context.Context, string) (*connectivity.Connection, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(testNewShowCommand(test.factory), "stripe-eu")
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), test.want), "error = %v", err)
		})
	}
}

func TestShowRecordedErrorsDoNotTurnSuccessfulReadIntoFailure(t *testing.T) {
	for _, state := range []string{"running", "error", "failed", "finished", "unknown"} {
		instance := instanceFixture("stripe-eu")
		instance.ConnectionStatus.IngestionStatus.State = ptr(connectivity.State(state))
		instance.ConnectionStatus.IngestionStatus.IngestionProgress = &connectivity.IngestionProgress{Current: 0}
		output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), "stripe-eu")
		require.NoError(t, err)
		require.Contains(t, output, "Last Error (may be historical)")
		require.Contains(t, output, "source temporarily unavailable")
		require.Equal(t, "0", ingestionProgress(instance.ConnectionStatus))
		instance.ConnectionStatus.IngestionStatus.IngestionProgress = nil
		require.Empty(t, ingestionProgress(instance.ConnectionStatus))
	}
}

func TestShowImportedConnectionJSONPreservesAbsentIdentities(t *testing.T) {
	instance := connectivity.Connection{Name: "imported"}
	command := testNewShowCommand(factoryWithInstance(instance))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err := executeCommand(command, "imported", "--output=json")
	require.NoError(t, err)
	require.JSONEq(t, `{"data":{"connection":{"name":"imported","configuration":{}}}}`, output)
	output, err = executeCommand(testNewShowCommand(factoryWithInstance(instance)), "imported")
	require.NoError(t, err)
	require.NotContains(t, output, "%!", "absent identity must be safe to render")
}

func TestShowRawLastErrorWithoutIngestion(t *testing.T) {
	const rawError = `{"errorCode":"provider_failed","details":{"request":9007199254740993}}`
	instance := instanceFixture("stripe-eu")
	instance.ConnectionStatus.IngestionStatus = nil
	instance.ConnectionStatus.LastError = stringPtr(rawError)
	output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), "stripe-eu")
	require.NoError(t, err)
	require.Contains(t, output, "Last Error (may be historical)")
	require.Contains(t, output, rawError)
	require.NotContains(t, output, "Service Error")
	command := testNewShowCommand(factoryWithInstance(instance))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err = executeCommand(command, "stripe-eu", "--output=json")
	require.NoError(t, err)
	var envelope struct {
		Data ShowStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Nil(t, envelope.Data.Connection.ConnectionStatus.IngestionStatus)
	require.Equal(t, rawError, *envelope.Data.Connection.ConnectionStatus.LastError)
}
