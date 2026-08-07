package instances

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type showInstanceClientMock struct {
	connectivityclient.Client
	get func(context.Context, string) (*connectivityclient.Instance, error)
}

func (m showInstanceClientMock) GetInstance(ctx context.Context, name string) (*connectivityclient.Instance, error) {
	return m.get(ctx, name)
}

func TestShowInstanceRendersMetadataDesiredStateLifecycleProgressAndRedactedConfigSources(t *testing.T) {
	created := time.Date(2026, time.August, 7, 10, 30, 0, 0, time.UTC)
	instance := instanceFixture("stripe-eu")
	instance.Metadata.Namespace = stringPtr("formance")
	instance.Metadata.ResourceVersion = stringPtr("42")
	instance.Metadata.UID = stringPtr("instance-uid")
	instance.Metadata.CreationTimestamp = &created
	instance.Metadata.Labels = map[string]string{"region": "eu"}
	instance.Metadata.Annotations = map[string]string{"owner": "platform"}
	mode := int32(420)
	instance.Spec.Config = &connectivityclient.InstanceConfig{
		Env: map[string]connectivityclient.EnvValue{
			"API_KEY": {Value: stringPtr("must-not-leak")},
			"TOKEN":   {SecretRef: &connectivityclient.KeyRef{Name: "plugin-secrets", Key: "token"}},
		},
		Files: []connectivityclient.FileMount{
			{Path: "/etc/plugin/config.yaml", ConfigMapRef: &connectivityclient.KeyRef{Name: "plugin-config", Key: "config.yaml"}, Mode: &mode},
			{Path: "/etc/plugin/inline.pem", Value: stringPtr("also-must-not-leak")},
		},
	}

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	for _, expected := range []string{
		"Information", "Name", "stripe-eu", "Namespace", "formance", "UID", "instance-uid", "Resource Version", "42",
		"Created At", created.Format(time.RFC3339), "Labels", "region=eu", "Annotations", "owner=platform",
		"Desired Specification", "Plugin", "stripe", "Version", "2.0.0", "Ledger", "main", "Poll Interval", "5s",
		"Lifecycle", "Resolved Image", "registry/plugin:2.0.0", "Plugin Address", "http://stripe.default.svc", "Phase", "Ready", "State", "Running",
		"Ingestion Progress", "Current Sequence", "42", "Source Tip Sequence", "48", "Last Error", "source temporarily unavailable", "Message", "retrying ingestion",
		"Configuration", "API_KEY", "environment", "inline", "TOKEN", "secret:plugin-secrets/token",
		"/etc/plugin/config.yaml", "file", "configmap:plugin-config/config.yaml", "420", "/etc/plugin/inline.pem",
	} {
		require.Contains(t, output, expected)
	}
	require.NotContains(t, output, "must-not-leak")
	require.NotContains(t, output, "also-must-not-leak")
}

func TestShowPlainOutputNeverPrintsInlineConfigValues(t *testing.T) {
	instance := instanceWithTwoFiles()
	instance.Spec.Config.Env = map[string]connectivityclient.EnvValue{
		"API_KEY": {Value: stringPtr("must-not-leak")},
	}
	command := NewShowCommand(factoryWithInstance(*instance))

	output, err := executeCommand(command, "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "API_KEY")
	require.Contains(t, output, "inline")
	require.NotContains(t, output, "must-not-leak")
	require.NotContains(t, output, "private config")
}

func TestShowPlainOutputOmitsUnsupportedStartSequence(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Spec.StartSequence = fctl.Ptr(int64(987654321))

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.NotContains(t, output, "Start Sequence")
	require.NotContains(t, output, "987654321")
}

func TestShowInstanceJSONPreservesCompleteModel(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Spec.Config = &connectivityclient.InstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"API_KEY": {Value: stringPtr("json-keeps-full-model")},
	}}
	command := NewShowCommand(factoryWithInstance(instance))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "--output", "json", "stripe-eu")

	require.NoError(t, err)
	var envelope struct {
		Data ShowStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, instance, envelope.Data.Instance)
}

func TestShowInstanceHandlesMissingStatusAndConfig(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Status = nil
	instance.Spec.Config = nil

	output, err := executeCommand(NewShowCommand(factoryWithInstance(instance)), "stripe-eu")

	require.NoError(t, err)
	require.Contains(t, output, "No configuration entries.")
}

func TestShowInstanceReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
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
			factory: factoryReturning(showInstanceClientMock{get: func(context.Context, string) (*connectivityclient.Instance, error) {
				return nil, errors.New("instance unavailable")
			}}),
			want: "instance unavailable",
		},
		"empty response": {
			factory: factoryReturning(showInstanceClientMock{get: func(context.Context, string) (*connectivityclient.Instance, error) {
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
