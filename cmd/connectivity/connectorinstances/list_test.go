package connectorinstances

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type listInstanceClientMock struct {
	connectivityclient.Client
	list func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error)
}

func (m listInstanceClientMock) ListConnectorInstances(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
	return m.list(ctx, options)
}

func TestListConnectorInstancesPassesFiltersAndPaginationAndRendersApprovedColumns(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	channelInstance := instanceFixture("stripe-eu")
	channelInstance.Spec.Version = nil
	channelInstance.Spec.Channel = stringPtr("stable")
	channelInstance.Status.ResolvedVersion = stringPtr("v1.3.0")
	client := listInstanceClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
		gotOptions = options
		return &connectivityclient.ConnectorInstanceList{
			Items:    []connectivityclient.ConnectorInstance{channelInstance},
			PageSize: 7,
			HasMore:  true,
			Next:     "next-page",
		}, nil
	}}

	output, err := executeCommand(NewListCommand(factoryReturning(client)), "--connector", "stripe", "--page-size", "7", "--cursor", "current-page")

	require.NoError(t, err)
	wantOptions := connectivityclient.ListOptions{
		Query:    `{"$match":{"connector":"stripe"}}`,
		PageSize: 7,
		Cursor:   "current-page",
	}
	require.Equal(t, wantOptions, gotOptions)
	for _, expected := range []string{
		"Name", "Connector", "Version", "Channel", "Resolved Version", "Ledger", "Phase", "State", "Current Sequence", "Source Tip Sequence", "Last Error",
		"stripe-eu", "stripe", "stable", "v1.3.0", "main", "Ready", "Running", "42", "48", "source temporarily unavailable",
		"HasMore", "true", "PageSize", "7", "Next", "next-page",
	} {
		require.Contains(t, output, expected)
	}
}

func TestListConnectorInstancesConjoinsConnectorAndFilterExpressions(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	client := listInstanceClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
		gotOptions = options
		return &connectivityclient.ConnectorInstanceList{Items: []connectivityclient.ConnectorInstance{}}, nil
	}}

	_, err := executeCommand(NewListCommand(factoryReturning(client)), "--filter", "phase=Running", "--connector", "stripe")

	require.NoError(t, err)
	want := `{"$and":[{"$match":{"phase":"Running"}},{"$match":{"connector":"stripe"}}]}`
	require.Equal(t, want, gotOptions.Query)
}

func TestListConnectorInstancesPassesRawQueryThrough(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	client := listInstanceClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
		gotOptions = options
		return &connectivityclient.ConnectorInstanceList{Items: []connectivityclient.ConnectorInstance{}}, nil
	}}
	raw := `{"$exists":{"channel":true}}`

	_, err := executeCommand(NewListCommand(factoryReturning(client)), "--query", raw)

	require.NoError(t, err)
	require.Equal(t, raw, gotOptions.Query)
}

func TestListConnectorInstancesRejectsRawQueryCombinedWithSugarFlags(t *testing.T) {
	client := listInstanceClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
		t.Fatal("ListConnectorInstances must not run with conflicting flags")
		return nil, nil
	}}

	_, err := executeCommand(NewListCommand(factoryReturning(client)), "--query", `{"$match":{"phase":"Running"}}`, "--connector", "stripe")

	require.Error(t, err)
	require.Contains(t, err.Error(), "--query cannot be combined with --filter")
}

func TestListConnectorInstancesWiresFilterAndConnectorCompletions(t *testing.T) {
	command := NewListCommand(factoryReturning(listInstanceClientMock{}))

	_, filterRegistered := command.GetFlagCompletionFunc("filter")
	require.True(t, filterRegistered, "--filter must complete from /_query/capabilities")
	_, connectorRegistered := command.GetFlagCompletionFunc("connector")
	require.True(t, connectorRegistered, "--connector must complete from the connector catalogue")
}

func TestListConnectorInstancesJSONPreservesCompleteModelsAndContinuation(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Metadata.Labels = map[string]string{"region": "eu"}
	instance.Spec.Config = &connectivityclient.ConnectorInstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"API_KEY": {Value: stringPtr("json-keeps-full-model")},
	}}
	client := listInstanceClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
		return &connectivityclient.ConnectorInstanceList{Items: []connectivityclient.ConnectorInstance{instance}, PageSize: 4, HasMore: true, Next: "next-page"}, nil
	}}
	command := NewListCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "--output", "json", "--page-size", "4")

	require.NoError(t, err)
	var envelope struct {
		Data ListStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, []connectivityclient.ConnectorInstance{instance}, envelope.Data.ConnectorInstances)
	require.True(t, envelope.Data.Cursor.HasMore)
	require.Equal(t, int64(4), envelope.Data.Cursor.PageSize)
	require.NotNil(t, envelope.Data.Cursor.Next)
	require.Equal(t, "next-page", *envelope.Data.Cursor.Next)
}

func TestConnectorInstanceRootRegistersReadCommandsAndAliases(t *testing.T) {
	command := NewCommand(factoryReturning(listInstanceClientMock{}), mockReadFile(nil), mockPathCompleter(nil))

	require.Equal(t, "connectorinstances", command.Use)
	require.Equal(t, []string{"connectorinstance", "instances", "instance", "ci"}, command.Aliases)
	wantAliases := map[string][]string{
		"list": {"ls", "l"},
		"show": {"get", "g", "sh", "s"},
	}
	for name, aliases := range wantAliases {
		child, _, err := command.Find([]string{name})
		require.NoError(t, err)
		require.Equal(t, name, child.Name())
		require.True(t, reflect.DeepEqual(aliases, child.Aliases), "%s aliases = %v", name, child.Aliases)
	}
}

func TestListConnectorInstancesReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
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
			factory: factoryReturning(listInstanceClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
				return nil, errors.New("connector instances unavailable")
			}}),
			want: "connector instances unavailable",
		},
		"empty response": {
			factory: factoryReturning(listInstanceClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(NewListCommand(test.factory))
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), test.want), "error = %v", err)
		})
	}
}
