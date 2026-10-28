package connections

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type showInstanceClientMock struct {
	connectivityclient.Client
	get func(context.Context, string) (*connectivityclient.ConnectorInstance, error)
}

func (m showInstanceClientMock) GetConnectorInstance(ctx context.Context, name string) (*connectivityclient.ConnectorInstance, error) {
	return m.get(ctx, name)
}

func TestShowRendersPortableStatusAndHidesConfigurationValues(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Configuration.Env = map[string]string{"TOKEN": "must-not-leak"}
	instance.Configuration.Files = map[string]string{"/etc/plugin/config.yaml": "also-must-not-leak"}
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
	instance.Configuration.Env = map[string]string{
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
	instance.Configuration.InitialCursor = []byte("opaque-initial-cursor")

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.NotContains(t, output, "Start Sequence")
	require.NotContains(t, output, "987654321")
}

func TestShowConnectorInstanceJSONPreservesCompleteModel(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	setInstanceConfig(&instance, &connectivityclient.ConnectorInstanceConfig{Env: map[string]string{
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
	require.Equal(t, instance, envelope.Data.ConnectorInstance)
}

func TestShowConnectorInstanceHandlesMissingStatusAndConfig(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Status = nil
	setInstanceConfig(&instance, nil)

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "No configuration entries.")
}

func TestShowConnectionRendersSuspensionAndOrderedProvenance(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Configuration.Suspend = fctl.Ptr(true)
	instance.Status.SuspendedBy = []connectivityclient.SuspensionSource{
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

func TestShowConnectorInstanceReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (connectivityclient.Client, error)
		want    string
	}{
		"missing factory": {factory: nil, want: "factory is required"},
		"factory": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return nil, errors.New("authentication failed")
			},
			want: "authentication failed",
		},
		"API": {
			factory: factoryReturning(showInstanceClientMock{get: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) {
				return nil, errors.New("connector instance unavailable")
			}}),
			want: "connector instance unavailable",
		},
		"empty response": {
			factory: factoryReturning(showInstanceClientMock{get: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) {
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
		instance.Status.Ingestion.State = stringPtr(state)
		instance.Status.Ingestion.Progress = &connectivityclient.IngestionProgress{Current: 0}
		instance.Status.Error = &connectivityclient.StatusError{Code: "failure_unknown", Message: "Failure provenance unavailable"}
		output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")
		require.NoError(t, err)
		require.Contains(t, output, "Last Error (may be historical)")
		require.Contains(t, output, "source temporarily unavailable")
		require.Contains(t, output, "failure_unknown")
		require.Equal(t, "0", ingestionProgress(instance.Status))
		instance.Status.Ingestion.Progress = nil
		require.Empty(t, ingestionProgress(instance.Status))
	}
}
