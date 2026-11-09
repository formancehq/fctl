package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

func versionsFixture() *connectorVersionPage {
	return &connectorVersionPage{Items: []connectivity.ConnectorVersionSummary{
		{Version: "1.0.0", Digest: fctl.Ptr("sha256:one")},
		{Version: "2.0.0"},
	}}
}

func TestShowConnectorRendersPublicFieldsVersionCountAndLatestVersionSchema(t *testing.T) {
	created := time.Date(2026, time.August, 7, 10, 30, 0, 0, time.UTC)
	connector := connectorFixture("stripe")
	connector.CreatedAt = &created

	var gotVersion string
	client := connectorTransportMock{
		get: func(_ context.Context, name string) (*connectivity.Connector, error) {
			if name != "stripe" {
				t.Fatalf("GetConnector name = %q, want stripe", name)
			}
			return &connector, nil
		},
		listVersions: func(_ context.Context, name string, options connectivityinternal.ListOptions) (*connectorVersionPage, error) {
			if name != "stripe" {
				t.Fatalf("ListConnectorVersions name = %q, want stripe", name)
			}
			if options.PageSize != 100 {
				t.Fatalf("ListConnectorVersions page size = %d, want 100", options.PageSize)
			}
			return versionsFixture(), nil
		},
		getVersion: func(_ context.Context, _, version string) (*connectivity.ConnectorVersion, error) {
			gotVersion = version
			return &connectivity.ConnectorVersion{
				Version: "2.0.0",
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
	command := testNewShowCommand(sdkFor(client))
	output, err := executeCommand(command, "stripe")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}

	if gotVersion != *connector.LatestVersion {
		t.Fatalf("GetConnectorVersion version = %q, want the catalogue version %q", gotVersion, *connector.LatestVersion)
	}
	for _, expected := range []string{
		"stripe", created.Format(time.RFC3339),
		"Connector display name", "Connector description", "public",
		"payments, webhooks", "Payments infrastructure",
		"Latest Version", "2.0.0",
		"Available Versions",
		"Configuration (2.0.0) · 2 fields, 1 required", "API_KEY", "yes", "secret", "API credential",
		"/etc/connector.yaml", "string", "Connector settings",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("plain output missing %q:\n%s", expected, output)
		}
	}
	for _, absent := range []string{"Digest", "Released", "sha256:one", "Namespace", "Resource Version", "UID", "Labels", "Annotations", "Image", "Phase", "Status Message"} {
		if strings.Contains(output, absent) {
			t.Errorf("public output must not include %q:\n%s", absent, output)
		}
	}
}

func TestShowConnectorFollowsVersionPagination(t *testing.T) {
	connector := connectorFixture("stripe")
	var gotCursors []string
	client := connectorTransportMock{
		get: func(context.Context, string) (*connectivity.Connector, error) {
			return &connector, nil
		},
		listVersions: func(_ context.Context, _ string, options connectivityinternal.ListOptions) (*connectorVersionPage, error) {
			gotCursors = append(gotCursors, options.Cursor)
			if options.Cursor == "" {
				return &connectorVersionPage{
					Items:   []connectivity.ConnectorVersionSummary{{Version: "1.0.0"}},
					HasMore: true,
					Next:    "page-two",
				}, nil
			}
			return &connectorVersionPage{
				Items: []connectivity.ConnectorVersionSummary{{Version: "2.0.0"}},
			}, nil
		},
	}

	output, err := executeCommand(testNewShowCommand(sdkFor(client)), "stripe")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}

	if !reflect.DeepEqual(gotCursors, []string{"", "page-two"}) {
		t.Fatalf("version list cursors = %#v, want the continuation to be followed", gotCursors)
	}
	require.Regexp(t, `Available Versions[^\n]*\b2\b`, output)

}

func TestShowConnectorWithoutVersionsReportsNoPublishedVersion(t *testing.T) {
	connector := connectorFixture("fresh")
	connector.LatestVersion = nil
	client := connectorTransportMock{
		get: func(context.Context, string) (*connectivity.Connector, error) {
			return &connector, nil
		},
		listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*connectorVersionPage, error) {
			return &connectorVersionPage{Items: []connectivity.ConnectorVersionSummary{}}, nil
		},
		getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
			t.Fatal("GetConnectorVersion must not run without a published version")
			return nil, nil
		},
	}

	output, err := executeCommand(testNewShowCommand(sdkFor(client)), "fresh")
	if err != nil {
		t.Fatalf("execute show command: %v", err)
	}
	if !strings.Contains(output, "No published version.") {
		t.Fatalf("output missing the no-version notice:\n%s", output)
	}
}

// A failed latest-version read is returned as the SDK error without inference.
func TestShowConnectorReturnsUnresolvableLatestError(t *testing.T) {
	connector := connectorFixture("pending")
	client := connectorTransportMock{
		get: func(context.Context, string) (*connectivity.Connector, error) {
			return &connector, nil
		},
		listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*connectorVersionPage, error) {
			return versionsFixture(), nil
		},
		getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
			return nil, &sdkerrors.SDKError{StatusCode: http.StatusNotFound, Body: `{"errorCode":"channel_empty","errorMessage":"No validated version"}`}
		},
	}

	output, err := executeCommand(testNewShowCommand(sdkFor(client)), "pending")
	var apiError *connectivity.Error
	if !errors.As(err, &apiError) {
		t.Fatalf("expected native Connectivity error, got %T: %v", err, err)
	}
	if err != apiError || apiError.ErrorCode == nil || *apiError.ErrorCode != "channel_empty" || apiError.ErrorMessage == nil || *apiError.ErrorMessage != "No validated version" {
		t.Fatalf("native SDK error was changed: %#v", err)
	}
	if strings.Contains(output, "No published version.") {
		t.Fatalf("an API failure must not render a catalogue notice: %s", output)
	}
}

func TestShowConnectorJSONPreservesCompleteModel(t *testing.T) {
	connector := connectorFixture("stripe")
	version := &connectivity.ConnectorVersion{
		Version:      "2.0.0",
		ConfigSchema: map[string]any{"type": "object"},
	}
	client := connectorTransportMock{
		get: func(context.Context, string) (*connectivity.Connector, error) {
			return &connector, nil
		},
		listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*connectorVersionPage, error) {
			return versionsFixture(), nil
		},
		getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
			return version, nil
		},
	}

	command := testNewShowCommand(sdkFor(client))
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

func TestShowConnectorReturnsTransportAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		client *formance.FormanceV1
		want   string
	}{
		"API": {
			client: sdkFor(connectorTransportMock{get: func(context.Context, string) (*connectivity.Connector, error) {
				return nil, errors.New("connector unavailable")
			}}),
			want: "connector unavailable",
		},
		"empty response": {
			client: sdkFor(connectorTransportMock{get: func(context.Context, string) (*connectivity.Connector, error) {
				return nil, nil
			}}),
			want: "unexpected end of JSON input",
		},
		"version list API": {
			client: sdkFor(connectorTransportMock{
				get: func(context.Context, string) (*connectivity.Connector, error) {
					connector := connectorFixture("stripe")
					return &connector, nil
				},
				listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*connectorVersionPage, error) {
					return nil, errors.New("catalog unavailable")
				},
			}),
			want: "catalog unavailable",
		},
		"latest version API": {
			client: sdkFor(connectorTransportMock{
				get: func(context.Context, string) (*connectivity.Connector, error) {
					connector := connectorFixture("stripe")
					return &connector, nil
				},
				listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*connectorVersionPage, error) {
					return versionsFixture(), nil
				},
				getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
					return nil, errors.New("catalog unavailable")
				},
			}),
			want: "catalog unavailable",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(testNewShowCommand(test.client), "stripe")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}

func TestConnectorSummaryDoesNotAdaptFlatSchema(t *testing.T) {
	rows := schemaFields(map[string]any{"properties": map[string]any{"OLD_TOKEN": map[string]any{"type": "string"}}})
	if len(rows) != 0 {
		t.Fatalf("flat schema must not become an environment section: %v", rows)
	}
}
