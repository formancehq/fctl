package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func TestListConnectorsPassesPaginationAndRendersApprovedColumnsAndContinuation(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	client := connectorTransportMock{list: func(_ context.Context, options connectivityinternal.ListOptions) (*connectorPage, error) {
		gotOptions = options
		return &connectorPage{
			Items:    []connectivity.Connector{connectorFixture("stripe")},
			PageSize: 7,
			HasMore:  true,
			Next:     "next-page",
		}, nil
	}}

	command := testNewListCommand(sdkFor(client))
	output, err := executeCommand(command, "--page-size", "7", "--cursor", "current-page")
	if err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	wantOptions := connectivityinternal.ListOptions{PageSize: 7, Cursor: "current-page"}
	if !reflect.DeepEqual(gotOptions, wantOptions) {
		t.Fatalf("ListConnectors options = %#v, want %#v", gotOptions, wantOptions)
	}
	for _, expected := range []string{
		"Name", "Latest Version", "Tagline",
		"stripe", "2.0.0", "Payments infrastructure",
		"Next page: --cursor next-page",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("plain output missing %q:\n%s", expected, output)
		}
	}
	// The description is deliberately absent: it is a full sentence, and pterm
	// widens a column to its content, so carrying it here wrapped every row.
	for _, absent := range []string{"Tags", "payments, webhooks", "Description", "Connector description", "Phase", "Status", "Image", "Namespace", "Resource Version"} {
		if strings.Contains(output, absent) {
			t.Errorf("plain output should not carry %q:\n%s", absent, output)
		}
	}
}

func TestListConnectorsBuildsQueryFromFilterFlags(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	client := connectorTransportMock{list: func(_ context.Context, options connectivityinternal.ListOptions) (*connectorPage, error) {
		gotOptions = options
		return &connectorPage{Items: []connectivity.Connector{}}, nil
	}}

	_, err := executeCommand(testNewListCommand(sdkFor(client)), "--filter", "catalog=ee", "--filter", "name=stripe")
	if err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	want := `{"$and":[{"$match":{"catalog":"ee"}},{"$match":{"name":"stripe"}}]}`
	if gotOptions.Query != want {
		t.Fatalf("ListConnectors query = %q, want %q", gotOptions.Query, want)
	}
}

func TestListConnectorsPassesRawQueryThrough(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	client := connectorTransportMock{list: func(_ context.Context, options connectivityinternal.ListOptions) (*connectorPage, error) {
		gotOptions = options
		return &connectorPage{Items: []connectivity.Connector{}}, nil
	}}
	raw := `{"$exists":{"catalog":false}}`

	_, err := executeCommand(testNewListCommand(sdkFor(client)), "--query", raw)
	if err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	if gotOptions.Query != raw {
		t.Fatalf("ListConnectors query = %q, want %q", gotOptions.Query, raw)
	}
}

func TestListConnectorsRejectsCombinedQueryAndFilter(t *testing.T) {
	client := connectorTransportMock{list: func(context.Context, connectivityinternal.ListOptions) (*connectorPage, error) {
		t.Fatal("ListConnectors must not run with conflicting flags")
		return nil, nil
	}}

	_, err := executeCommand(testNewListCommand(sdkFor(client)), "--query", `{"$match":{"catalog":"ee"}}`, "--filter", "name=stripe")

	if err == nil || !strings.Contains(err.Error(), "--query cannot be combined with --filter") {
		t.Fatalf("error = %v, want the conflicting flags error", err)
	}
}

func TestListConnectorsJSONPreservesCompleteModelsAndContinuation(t *testing.T) {
	connector := connectorFixture("stripe")
	connector.ConnectorBranding.AccentColor = fctl.Ptr("#123456")
	client := connectorTransportMock{list: func(_ context.Context, _ connectivityinternal.ListOptions) (*connectorPage, error) {
		return &connectorPage{Items: []connectivity.Connector{connector}, PageSize: 4, HasMore: true, Next: "next-page"}, nil
	}}

	command := testNewListCommand(sdkFor(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err := executeCommand(command, "--output", "json", "--page-size", "4")
	if err != nil {
		t.Fatalf("execute JSON list command: %v", err)
	}

	var envelope struct {
		Data ListStore `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}
	if !reflect.DeepEqual(envelope.Data.Connectors, []connectivity.Connector{connector}) {
		t.Fatalf("JSON connectors = %#v, want complete model %#v", envelope.Data.Connectors, connector)
	}
	if !envelope.Data.Cursor.HasMore || envelope.Data.Cursor.PageSize != 4 || envelope.Data.Cursor.Next == nil || *envelope.Data.Cursor.Next != "next-page" {
		t.Fatalf("JSON cursor = %#v, want continuation and served page size", envelope.Data.Cursor)
	}
}

func TestListConnectorsReturnsTransportAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		client *formance.FormanceV1
		want   string
	}{
		"API": {
			client: sdkFor(connectorTransportMock{list: func(context.Context, connectivityinternal.ListOptions) (*connectorPage, error) {
				return nil, errors.New("catalog unavailable")
			}}),
			want: "catalog unavailable",
		},
		"empty response": {
			client: sdkFor(connectorTransportMock{list: func(context.Context, connectivityinternal.ListOptions) (*connectorPage, error) {
				return nil, nil
			}}),
			want: "unexpected end of JSON input",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(testNewListCommand(test.client))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}
