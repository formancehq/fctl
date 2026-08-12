package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func versionsFixture() *connectivityclient.ConnectorVersionList {
	return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{
		{Version: "1.0.0", Image: "registry/connector:1", Digest: fctl.Ptr("sha256:one")},
		{Version: "2.0.0", Image: "registry/connector:2"},
	}}
}

func TestShowConnectorRendersMetadataVersionsTagsStatusAndNewestVersionSchema(t *testing.T) {
	created := time.Date(2026, time.August, 7, 10, 30, 0, 0, time.UTC)
	connector := connectorFixture("stripe")
	connector.Metadata.Namespace = fctl.Ptr("formance")
	connector.Metadata.ResourceVersion = fctl.Ptr("42")
	connector.Metadata.UID = fctl.Ptr("connector-uid")
	connector.Metadata.CreationTimestamp = &created
	connector.Metadata.Labels = map[string]string{"region": "eu"}
	connector.Metadata.Annotations = map[string]string{"owner": "platform"}
	connector.Status.Message = fctl.Ptr("Catalog entry is healthy")

	var gotVersion string
	client := connectorClientMock{
		get: func(_ context.Context, name string) (*connectivityclient.Connector, error) {
			if name != "stripe" {
				t.Fatalf("GetConnector name = %q, want stripe", name)
			}
			return &connector, nil
		},
		listVersions: func(_ context.Context, name string) (*connectivityclient.ConnectorVersionList, error) {
			if name != "stripe" {
				t.Fatalf("ListConnectorVersions name = %q, want stripe", name)
			}
			return versionsFixture(), nil
		},
		getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
			gotVersion = version
			return &connectivityclient.ConnectorVersion{
				Version: version,
				Image:   "registry/connector:2",
				ConfigSchema: map[string]any{
					"type": "object",
					"env": map[string]any{
						"type":       "object",
						"required":   []any{"API_KEY"},
						"properties": map[string]any{"API_KEY": map[string]any{"type": "string", "format": "password", "description": "API credential"}},
					},
					"files": map[string]any{
						"type":       "object",
						"properties": map[string]any{"/etc/connector.yaml": map[string]any{"type": "string", "description": "Connector settings"}},
					},
				},
			}, nil
		},
	}
	command := NewShowCommand(factoryReturning(client))
	output, err := executeCommand(command, "stripe")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}

	if gotVersion != "2.0.0" {
		t.Fatalf("GetConnectorVersion version = %q, want the newest published version", gotVersion)
	}
	for _, expected := range []string{
		"stripe", "formance", "42", "connector-uid", created.Format(time.RFC3339), "region=eu", "owner=platform",
		"Connector display name", "Connector description", "registry/connector", "public",
		"payments, webhooks", "Ready", "Catalog entry is healthy",
		"1.0.0", "sha256:one", "2.0.0",
		"Configuration Schema (2.0.0)", "API_KEY", "environment", "required", "password", "API credential",
		"/etc/connector.yaml", "file", "Connector settings",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("plain output missing %q:\n%s", expected, output)
		}
	}
}

func TestShowConnectorWithoutVersionsReportsNoPublishedVersion(t *testing.T) {
	connector := connectorFixture("fresh")
	client := connectorClientMock{
		get: func(context.Context, string) (*connectivityclient.Connector, error) {
			return &connector, nil
		},
		listVersions: func(context.Context, string) (*connectivityclient.ConnectorVersionList, error) {
			return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{}}, nil
		},
		getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
			t.Fatal("GetConnectorVersion must not run without a published version")
			return nil, nil
		},
	}

	output, err := executeCommand(NewShowCommand(factoryReturning(client)), "fresh")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}
	if !strings.Contains(output, "No published version.") {
		t.Fatalf("output missing the no-version notice:\n%s", output)
	}
}

func TestShowConnectorJSONPreservesCompleteModel(t *testing.T) {
	connector := connectorFixture("stripe")
	version := &connectivityclient.ConnectorVersion{
		Version:      "2.0.0",
		Image:        "registry/connector:2",
		ConfigSchema: map[string]any{"type": "object"},
	}
	client := connectorClientMock{
		get: func(context.Context, string) (*connectivityclient.Connector, error) {
			return &connector, nil
		},
		listVersions: func(context.Context, string) (*connectivityclient.ConnectorVersionList, error) {
			return versionsFixture(), nil
		},
		getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
			return version, nil
		},
	}

	command := NewShowCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err := executeCommand(command, "--output", "json", "stripe")
	if err != nil {
		t.Fatalf("execute JSON show command: %v", err)
	}

	var envelope struct {
		Data ShowStore `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}
	if !reflect.DeepEqual(envelope.Data.Connector, connector) {
		t.Fatalf("JSON connector = %#v, want complete model %#v", envelope.Data.Connector, connector)
	}
	if !reflect.DeepEqual(envelope.Data.Versions, versionsFixture().Items) {
		t.Fatalf("JSON versions = %#v, want the complete version list", envelope.Data.Versions)
	}
	if !reflect.DeepEqual(envelope.Data.Version, version) {
		t.Fatalf("JSON version = %#v, want complete model %#v", envelope.Data.Version, version)
	}
}

func TestShowConnectorReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (connectivityclient.Client, error)
		want    string
	}{
		"factory": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return nil, errors.New("authentication failed")
			},
			want: "authentication failed",
		},
		"API": {
			factory: factoryReturning(connectorClientMock{get: func(context.Context, string) (*connectivityclient.Connector, error) {
				return nil, errors.New("connector unavailable")
			}}),
			want: "connector unavailable",
		},
		"empty response": {
			factory: factoryReturning(connectorClientMock{get: func(context.Context, string) (*connectivityclient.Connector, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
		"version list API": {
			factory: factoryReturning(connectorClientMock{
				get: func(context.Context, string) (*connectivityclient.Connector, error) {
					connector := connectorFixture("stripe")
					return &connector, nil
				},
				listVersions: func(context.Context, string) (*connectivityclient.ConnectorVersionList, error) {
					return nil, errors.New("catalog unavailable")
				},
			}),
			want: "catalog unavailable",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(NewShowCommand(test.factory), "stripe")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}

func TestShowConnectorSummarizesLegacyFlatSchema(t *testing.T) {
	connector := connectorFixture("legacy")
	connector.Status = nil
	client := connectorClientMock{
		get: func(context.Context, string) (*connectivityclient.Connector, error) {
			return &connector, nil
		},
		listVersions: func(context.Context, string) (*connectivityclient.ConnectorVersionList, error) {
			return versionsFixture(), nil
		},
		getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
			return &connectivityclient.ConnectorVersion{
				Version: version,
				Image:   "registry/connector:2",
				ConfigSchema: map[string]any{
					"type":     "object",
					"required": []string{"ENDPOINT"},
					"properties": map[string]any{
						"ENDPOINT": map[string]any{"type": "string", "description": "Service endpoint"},
					},
				},
			}, nil
		},
	}

	output, err := executeCommand(NewShowCommand(factoryReturning(client)), "legacy")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}
	for _, expected := range []string{"ENDPOINT", "environment", "required", "string", "Service endpoint"} {
		if !strings.Contains(output, expected) {
			t.Errorf("legacy schema output missing %q:\n%s", expected, output)
		}
	}
}
