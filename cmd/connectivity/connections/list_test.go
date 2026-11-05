package connections

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type listInstanceClientMock struct {
	list func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error)
}

func TestListConnectionsPassesFiltersAndPaginationAndRendersApprovedColumns(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	channelInstance := instanceFixture("stripe-eu")
	channelInstance.ConnectionConfiguration.Version = nil
	channelInstance.ConnectionConfiguration.Channel = ptr(connectivity.Channel("stable"))
	channelInstance.ConnectionStatus.AppliedVersion = stringPtr("v1.3.0")
	client := listInstanceClientMock{list: func(_ context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
		gotOptions = options
		return &testConnectionList{
			Items:    []connectivity.Connection{channelInstance},
			PageSize: 7,
			HasMore:  true,
			Next:     "next-page",
		}, nil
	}}

	output, err := executeCommand(testNewListCommand(factoryReturning(client)), "--connector", "stripe", "--page-size", "7", "--cursor", "current-page")

	require.NoError(t, err)
	wantOptions := connectivityinternal.ListOptions{
		Query:    `{"$match":{"connector":"stripe"}}`,
		PageSize: 7,
		Cursor:   "current-page",
	}
	require.Equal(t, wantOptions, gotOptions)
	for _, expected := range []string{
		"Name", "Connector", "Version", "Channel", "Applied Version", "Ledger", "Lifecycle", "Ingestion State", "Progress Current", "Last Error (may be historical)",
		"stripe-eu", "stripe", "stable", "v1.3.0", "main", "active", "running", "42", "source temporarily unavailable",
		"HasMore", "true", "PageSize", "7", "Next", "next-page",
	} {
		require.Contains(t, output, expected)
	}
}

func TestListConnectionsConjoinsConnectorAndFilterExpressions(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	client := listInstanceClientMock{list: func(_ context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
		gotOptions = options
		return &testConnectionList{Items: []connectivity.Connection{}}, nil
	}}

	_, err := executeCommand(testNewListCommand(factoryReturning(client)), "--filter", "lifecycle=active", "--connector", "stripe")

	require.NoError(t, err)
	want := `{"$and":[{"$match":{"lifecycle":"active"}},{"$match":{"connector":"stripe"}}]}`
	require.Equal(t, want, gotOptions.Query)
}

func TestListConnectionsPassesRawQueryThrough(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	client := listInstanceClientMock{list: func(_ context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
		gotOptions = options
		return &testConnectionList{Items: []connectivity.Connection{}}, nil
	}}
	raw := `{"$exists":{"channel":true}}`

	_, err := executeCommand(testNewListCommand(factoryReturning(client)), "--query", raw)

	require.NoError(t, err)
	require.Equal(t, raw, gotOptions.Query)
}

func TestListConnectionsRejectsRawQueryCombinedWithSugarFlags(t *testing.T) {
	client := listInstanceClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
		t.Fatal("ListConnections must not run with conflicting flags")
		return nil, nil
	}}

	_, err := executeCommand(testNewListCommand(factoryReturning(client)), "--query", `{"$match":{"lifecycle":"active"}}`, "--connector", "stripe")

	require.Error(t, err)
	require.Contains(t, err.Error(), "--query cannot be combined with --filter")
}

func TestListConnectionsWiresFilterAndConnectorCompletions(t *testing.T) {
	command := testNewListCommand(factoryReturning(listInstanceClientMock{}))

	_, filterRegistered := command.GetFlagCompletionFunc("filter")
	require.True(t, filterRegistered, "--filter must complete from /_query/capabilities")
	_, connectorRegistered := command.GetFlagCompletionFunc("connector")
	require.True(t, connectorRegistered, "--connector must complete from the connector catalogue")
}

func TestListConnectionsJSONPreservesCompleteModelsAndContinuation(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	setInstanceConfig(&instance, &connectivity.ConnectionConfiguration{Env: map[string]string{
		"API_KEY": "json-keeps-full-model",
	}})
	client := listInstanceClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
		return &testConnectionList{Items: []connectivity.Connection{instance}, PageSize: 4, HasMore: true, Next: "next-page"}, nil
	}}
	command := testNewListCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "--output", "json", "--page-size", "4")

	require.NoError(t, err)
	var envelope struct {
		Data ListStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, []connectivity.Connection{instance}, envelope.Data.Connections)
	require.True(t, envelope.Data.Cursor.HasMore)
	require.Equal(t, int64(4), envelope.Data.Cursor.PageSize)
	require.NotNil(t, envelope.Data.Cursor.Next)
	require.Equal(t, "next-page", *envelope.Data.Cursor.Next)
}

func TestConnectionRootRegistersReadCommandsAndAliases(t *testing.T) {
	command := testNewCommand(factoryReturning(listInstanceClientMock{}), mockReadFile(nil), mockPathCompleter(nil))

	require.Equal(t, "connections", command.Use)
	require.Equal(t, []string{"connection"}, command.Aliases)
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

func TestListConnectionsReturnsAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (*sdk.FormanceV1, error)
		want    string
	}{
		"API": {
			factory: factoryReturning(listInstanceClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
				return nil, errors.New("connections unavailable")
			}}),
			want: "connections unavailable",
		},
		"empty response": {
			factory: factoryReturning(listInstanceClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(testNewListCommand(test.factory))
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), test.want), "error = %v", err)
		})
	}
}

func TestListRendersRawLastErrorWithoutIngestion(t *testing.T) {
	const rawError = `{"errorCode":"provider_failed","details":"still historical"}`
	instance := instanceFixture("stripe-eu")
	instance.ConnectionStatus.IngestionStatus = nil
	instance.ConnectionStatus.LastError = stringPtr(rawError)
	client := listInstanceClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
		return &testConnectionList{Items: []connectivity.Connection{instance}}, nil
	}}
	output, err := executeCommand(testNewListCommand(factoryReturning(client)))
	require.NoError(t, err)
	require.Contains(t, output, "Last Error (may be historical)")
	require.Contains(t, output, rawError)
	require.NotContains(t, output, "Service Error")
}
