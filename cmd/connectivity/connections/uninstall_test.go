package connections

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type uninstallClientMock struct {
	connectivityinternal.API
	listInstances func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error)
	delete        func(context.Context, string) error
}

func (m uninstallClientMock) ListConnections(ctx context.Context, request operations.ListConnectionsRequest, security *operations.ListConnectionsSecurity, opts ...operations.Option) (*operations.ListConnectionsResponse, error) {
	data, err := m.listInstances(ctx, listOptions(request.PageSize, request.Cursor, request.Query))
	response := &operations.ListConnectionsResponse{StatusCode: 200}
	if data != nil {
		response.ConnectionCursor = &connectivity.ConnectionCursor{Cursor: connectivity.Cursor{Data: data.Items, HasMore: data.HasMore, Next: connectivityinternal.OptionalString(data.Next), PageSize: int(data.PageSize)}}
	}
	return response, err
}

func (m uninstallClientMock) DeleteConnection(ctx context.Context, request operations.DeleteConnectionRequest, security *operations.DeleteConnectionSecurity, opts ...operations.Option) (*operations.DeleteConnectionResponse, error) {
	err := m.delete(ctx, request.Name)
	return &operations.DeleteConnectionResponse{StatusCode: 204}, err
}

func TestUninstallDeletesConfirmedConnectionAndRendersSuccess(t *testing.T) {
	var deletedName string
	client := uninstallClientMock{delete: func(_ context.Context, name string) error {
		deletedName = name
		return nil
	}}

	output, err := executeCommand(NewUninstallCommand(factoryReturning(client)), "stripe-eu", "--confirm")

	require.NoError(t, err)
	require.Equal(t, "stripe-eu", deletedName)
	require.Equal(t, `Connection "stripe-eu" uninstalled.`, strings.TrimSpace(output))
}

func TestUninstallConfirmationRejectionPreventsDelete(t *testing.T) {
	deleted := false
	client := uninstallClientMock{delete: func(context.Context, string) error {
		deleted = true
		return errors.New("DeleteConnection must not be called")
	}}
	controller := NewUninstallController(factoryReturning(client))
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }

	_, err := controller.Run(&cobra.Command{}, []string{"stripe-eu"})

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.False(t, deleted)
}

func TestUninstallPreservesDeleteAPIError(t *testing.T) {
	apiError := &connectivityinternal.APIError{StatusCode: 404, Code: "CONNECTORINSTANCE_NOT_FOUND", Message: "missing"}
	client := uninstallClientMock{delete: func(context.Context, string) error { return apiError }}

	_, err := executeCommand(NewUninstallCommand(factoryReturning(client)), "stripe-eu", "--confirm")

	require.ErrorIs(t, err, apiError)
}

func TestUninstallRegistersAliasesRootAndConnectionCompletion(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	client := uninstallClientMock{
		listInstances: func(ctx context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
			require.True(t, connectivityinternal.IsNonInteractive(ctx))
			require.Equal(t, connectivityinternal.ListOptions{PageSize: 100}, options)
			return &testConnectionList{Items: []connectivity.Connection{instance}}, nil
		},
	}
	command := NewUninstallCommand(factoryReturning(client))

	require.Equal(t, "uninstall <connection>", command.Use)
	require.Equal(t, []string{"delete", "remove", "rm", "u"}, command.Aliases)
	require.Error(t, command.Args(command, nil))
	require.NoError(t, command.Args(command, []string{"stripe-eu"}))
	require.Error(t, command.Args(command, []string{"stripe-eu", "extra"}))
	candidates, directive := command.ValidArgsFunction(command, nil, "stripe")
	require.Equal(t, []string{"stripe-eu\tstripe · main"}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	root := NewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	child, _, err := root.Find([]string{"uninstall"})
	require.NoError(t, err)
	require.Equal(t, "uninstall", child.Name())
	require.True(t, reflect.DeepEqual([]string{"delete", "remove", "rm", "u"}, child.Aliases))
}

func TestUninstallInvalidArgumentsDoNotDelete(t *testing.T) {
	usedFactory := false
	factory := func(*cobra.Command) (connectivityinternal.API, error) {
		usedFactory = true
		return nil, errors.New("factory must not be used")
	}

	_, err := executeCommand(NewUninstallCommand(factory), "stripe-eu", "extra", "--confirm")

	require.Error(t, err)
	require.False(t, usedFactory)
}
