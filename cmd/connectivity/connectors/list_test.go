package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestListConnectorsPassesPaginationAndRendersApprovedColumnsAndContinuation(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	client := connectorClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
		gotOptions = options
		return &connectivityclient.ConnectorList{
			Items:    []connectivityclient.Connector{connectorFixture("stripe")},
			PageSize: 7,
			HasMore:  true,
			Next:     "next-page",
		}, nil
	}}

	command := NewListCommand(factoryReturning(client))
	output, err := executeCommand(command, "--page-size", "7", "--cursor", "current-page")
	if err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	wantOptions := connectivityclient.ListOptions{PageSize: 7, Cursor: "current-page"}
	if !reflect.DeepEqual(gotOptions, wantOptions) {
		t.Fatalf("ListConnectors options = %#v, want %#v", gotOptions, wantOptions)
	}
	for _, expected := range []string{
		"Name", "Display Name", "Description", "Tags", "Phase",
		"stripe", "Connector display name", "Connector description", "payments, webhooks", "Ready",
		"HasMore", "true", "PageSize", "7", "Next", "next-page",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("plain output missing %q:\n%s", expected, output)
		}
	}
}

func TestListConnectorsBuildsQueryFromFilterFlags(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	client := connectorClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
		gotOptions = options
		return &connectivityclient.ConnectorList{Items: []connectivityclient.Connector{}}, nil
	}}

	_, err := executeCommand(NewListCommand(factoryReturning(client)), "--filter", "catalog=ee", "--filter", "phase=Ready")
	if err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	want := `{"$and":[{"$match":{"catalog":"ee"}},{"$match":{"phase":"Ready"}}]}`
	if gotOptions.Query != want {
		t.Fatalf("ListConnectors query = %q, want %q", gotOptions.Query, want)
	}
}

func TestListConnectorsPassesRawQueryThrough(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	client := connectorClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
		gotOptions = options
		return &connectivityclient.ConnectorList{Items: []connectivityclient.Connector{}}, nil
	}}
	raw := `{"$exists":{"catalog":false}}`

	_, err := executeCommand(NewListCommand(factoryReturning(client)), "--query", raw)
	if err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	if gotOptions.Query != raw {
		t.Fatalf("ListConnectors query = %q, want %q", gotOptions.Query, raw)
	}
}

func TestListConnectorsRejectsCombinedQueryAndFilter(t *testing.T) {
	client := connectorClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
		t.Fatal("ListConnectors must not run with conflicting flags")
		return nil, nil
	}}

	_, err := executeCommand(NewListCommand(factoryReturning(client)), "--query", `{"$match":{"catalog":"ee"}}`, "--filter", "phase=Ready")

	if err == nil || !strings.Contains(err.Error(), "--query cannot be combined with --filter") {
		t.Fatalf("error = %v, want the conflicting flags error", err)
	}
}

func TestListConnectorsJSONPreservesCompleteModelsAndContinuation(t *testing.T) {
	connector := connectorFixture("stripe")
	connector.Metadata.Labels = map[string]string{"region": "eu"}
	client := connectorClientMock{list: func(_ context.Context, _ connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
		return &connectivityclient.ConnectorList{Items: []connectivityclient.Connector{connector}, PageSize: 4, HasMore: true, Next: "next-page"}, nil
	}}

	command := NewListCommand(factoryReturning(client))
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
	if !reflect.DeepEqual(envelope.Data.Connectors, []connectivityclient.Connector{connector}) {
		t.Fatalf("JSON connectors = %#v, want complete model %#v", envelope.Data.Connectors, connector)
	}
	if !envelope.Data.Cursor.HasMore || envelope.Data.Cursor.PageSize != 4 || envelope.Data.Cursor.Next == nil || *envelope.Data.Cursor.Next != "next-page" {
		t.Fatalf("JSON cursor = %#v, want continuation and served page size", envelope.Data.Cursor)
	}
}

func TestConnectorRootRegistersApprovedCommandsAndAliases(t *testing.T) {
	command := NewCommand(factoryReturning(connectorClientMock{}))
	if command.Use != "connectors" || !reflect.DeepEqual(command.Aliases, []string{"connector", "c"}) {
		t.Fatalf("connector root = %q aliases %v", command.Use, command.Aliases)
	}

	wantAliases := map[string][]string{
		"list":   {"ls", "l"},
		"show":   {"get", "g", "sh", "s"},
		"facets": {"facet", "f"},
	}
	for name, aliases := range wantAliases {
		child, _, err := command.Find([]string{name})
		if err != nil {
			t.Fatalf("find %s command: %v", name, err)
		}
		if child.Name() != name || !reflect.DeepEqual(child.Aliases, aliases) {
			t.Errorf("%s aliases = %v, want %v", name, child.Aliases, aliases)
		}
	}
}

func TestListConnectorsReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
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
			factory: factoryReturning(connectorClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
				return nil, errors.New("catalog unavailable")
			}}),
			want: "catalog unavailable",
		},
		"empty response": {
			factory: factoryReturning(connectorClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(NewListCommand(test.factory))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}
