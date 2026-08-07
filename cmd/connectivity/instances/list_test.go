package instances

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
	list func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error)
}

func (m listInstanceClientMock) ListInstances(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
	return m.list(ctx, options)
}

func TestListInstancesPassesFiltersAndPaginationAndRendersApprovedColumns(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	client := listInstanceClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
		gotOptions = options
		return &connectivityclient.InstanceList{
			Items:    []connectivityclient.Instance{instanceFixture("stripe-eu")},
			Continue: "next-page",
		}, nil
	}}

	output, err := executeCommand(NewListCommand(factoryReturning(client)), "--plugin", "stripe", "--page-size", "7", "--cursor", "current-page")

	require.NoError(t, err)
	require.Equal(t, connectivityclient.ListOptions{Plugin: "stripe", Limit: 7, Continue: "current-page"}, gotOptions)
	for _, expected := range []string{
		"Name", "Plugin", "Version", "Ledger", "Phase", "State", "Current Sequence", "Source Tip Sequence", "Last Error",
		"stripe-eu", "stripe", "2.0.0", "main", "Ready", "Running", "42", "48", "source temporarily unavailable",
		"HasMore", "true", "PageSize", "7", "Next", "next-page",
	} {
		require.Contains(t, output, expected)
	}
}

func TestListInstancesJSONPreservesCompleteModelsAndContinuation(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	instance.Metadata.Labels = map[string]string{"region": "eu"}
	instance.Spec.Config = &connectivityclient.InstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"API_KEY": {Value: stringPtr("json-keeps-full-model")},
	}}
	client := listInstanceClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
		return &connectivityclient.InstanceList{Items: []connectivityclient.Instance{instance}, Continue: "next-page"}, nil
	}}
	command := NewListCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "--output", "json", "--page-size", "4")

	require.NoError(t, err)
	var envelope struct {
		Data ListStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, []connectivityclient.Instance{instance}, envelope.Data.Instances)
	require.True(t, envelope.Data.Cursor.HasMore)
	require.Equal(t, int64(4), envelope.Data.Cursor.PageSize)
	require.NotNil(t, envelope.Data.Cursor.Next)
	require.Equal(t, "next-page", *envelope.Data.Cursor.Next)
}

func TestInstanceRootRegistersReadCommandsAndAliases(t *testing.T) {
	command := NewCommand(factoryReturning(listInstanceClientMock{}), mockReadFile(nil), mockPathCompleter(nil))

	require.Equal(t, "instances", command.Use)
	require.Equal(t, []string{"instance", "i"}, command.Aliases)
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

func TestListInstancesReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
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
			factory: factoryReturning(listInstanceClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
				return nil, errors.New("instances unavailable")
			}}),
			want: "instances unavailable",
		},
		"empty response": {
			factory: factoryReturning(listInstanceClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
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
