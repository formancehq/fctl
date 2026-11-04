package connections

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type showInstanceClientMock struct {
	connectivityinternal.API
	get func(context.Context, string) (*connectivity.Connection, error)
}

func (m showInstanceClientMock) GetConnection(ctx context.Context, request operations.GetConnectionRequest, security *operations.GetConnectionSecurity, opts ...operations.Option) (*operations.GetConnectionResponse, error) {
	data, err := m.get(ctx, request.Name)
	return &operations.GetConnectionResponse{StatusCode: 200, Connection: data}, err
}

func TestShowRendersPortableStatusAndHidesConfigurationValues(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.ConnectionConfiguration.Env = map[string]string{"TOKEN": "must-not-leak"}
	instance.ConnectionConfiguration.Files = map[string]string{"/etc/plugin/config.yaml": "also-must-not-leak"}
	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")
	require.NoError(t, err)
	for _, value := range []string{"stripe-eu", "stripe", "main", "Desired Configuration", "Applied Version", "2.0.0", "active", "running", "42", "Last Error (may be historical)", "source temporarily unavailable", "TOKEN", "/etc/plugin/config.yaml", "literal"} {
		require.Contains(t, output, value)
	}
	for _, value := range []string{"must-not-leak", "also-must-not-leak", "Namespace", "UID", "Resource Version", "Conditions", "Resolved Image", "Source Tip Sequence"} {
		require.NotContains(t, output, value)
	}
}

func TestShowPlainOutputNeverPrintsInlineConfigValues(t *testing.T) {
	instance := instanceWithTwoFiles()
	instance.ConnectionConfiguration.Env = map[string]string{
		"API_KEY": "must-not-leak",
	}
	command := NewShowCommand(factoryWithInstance(*instance))

	output, err := executeCommand(command, "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "API_KEY")
	require.Contains(t, output, "literal")
	require.NotContains(t, output, "must-not-leak")
	require.NotContains(t, output, "private config")
}

func TestShowPlainOutputOmitsUnsupportedStartSequence(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.ConnectionConfiguration.InitialCursor = stringPtr("b3BhcXVlLWluaXRpYWwtY3Vyc29y")

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.NotContains(t, output, "Start Sequence")
	require.NotContains(t, output, "987654321")
}

func TestShowConnectionJSONPreservesCompleteModel(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	setInstanceConfig(&instance, &connectivity.ConnectionConfiguration{Env: map[string]string{
		"API_KEY": "json-keeps-full-model",
	}})
	command := NewShowCommand(factoryWithInstance(instance))
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

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

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

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "Suspend")
	require.Contains(t, output, "true")
	require.NotContains(t, output, "Replicas")
	require.Contains(t, output, "Suspended by")
	require.Contains(t, output, "request/stripe-eu, policy/maintenance")
}

func TestShowConnectionReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (connectivityinternal.API, error)
		want    string
	}{
		"missing factory": {factory: nil, want: "factory is required"},
		"factory": {
			factory: func(*cobra.Command) (connectivityinternal.API, error) {
				return nil, errors.New("authentication failed")
			},
			want: "authentication failed",
		},
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
			_, err := executeCommand(NewShowCommand(test.factory), "stripe-eu")
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
		instance.ConnectionStatus.StatusError = &connectivity.StatusError{Code: "failure_unknown", Message: "Failure provenance unavailable"}
		output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")
		require.NoError(t, err)
		require.Contains(t, output, "Last Error (may be historical)")
		require.Contains(t, output, "source temporarily unavailable")
		require.Contains(t, output, "failure_unknown")
		require.Equal(t, "0", ingestionProgress(instance.ConnectionStatus))
		instance.ConnectionStatus.IngestionStatus.IngestionProgress = nil
		require.Empty(t, ingestionProgress(instance.ConnectionStatus))
	}
}

func TestShowImportedConnectionJSONPreservesAbsentIdentities(t *testing.T) {
	instance := connectivity.Connection{Name: "imported"}
	command := NewShowCommand(factoryWithInstance(instance))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err := executeCommand(command, "imported", "--output=json")
	require.NoError(t, err)
	require.JSONEq(t, `{"data":{"connection":{"name":"imported","configuration":{}}}}`, output)
	output, err = executeCommand(NewShowCommand(factoryWithInstance(instance)), "imported")
	require.NoError(t, err)
	require.NotContains(t, output, "%!", "absent identity must be safe to render")
}
