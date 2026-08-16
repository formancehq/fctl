package connectorinstances

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type uninstallClientMock struct {
	connectivityclient.Client
	listInstances func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error)
	delete        func(context.Context, string) error
}

func (m uninstallClientMock) ListConnectorInstances(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
	return m.listInstances(ctx, options)
}

func (m uninstallClientMock) DeleteConnectorInstance(ctx context.Context, name string) error {
	return m.delete(ctx, name)
}

func TestUninstallDeletesConfirmedConnectorInstanceAndRendersSuccess(t *testing.T) {
	var deletedName string
	client := uninstallClientMock{delete: func(_ context.Context, name string) error {
		deletedName = name
		return nil
	}}

	output, err := executeCommand(NewUninstallCommand(factoryReturning(client)), "stripe-eu", "--confirm")

	require.NoError(t, err)
	require.Equal(t, "stripe-eu", deletedName)
	require.Equal(t, `Connector instance "stripe-eu" uninstalled.`, strings.TrimSpace(output))
}

func TestUninstallConfirmationRejectionPreventsDelete(t *testing.T) {
	deleted := false
	client := uninstallClientMock{delete: func(context.Context, string) error {
		deleted = true
		return errors.New("DeleteConnectorInstance must not be called")
	}}
	controller := NewUninstallController(factoryReturning(client))
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }

	_, err := controller.Run(&cobra.Command{}, []string{"stripe-eu"})

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.False(t, deleted)
}

func TestUninstallPreservesDeleteAPIError(t *testing.T) {
	apiError := &connectivityclient.APIError{StatusCode: 404, Code: "CONNECTORINSTANCE_NOT_FOUND", Message: "missing"}
	client := uninstallClientMock{delete: func(context.Context, string) error { return apiError }}

	_, err := executeCommand(NewUninstallCommand(factoryReturning(client)), "stripe-eu", "--confirm")

	require.ErrorIs(t, err, apiError)
}

func TestUninstallRegistersAliasesRootAndConnectorInstanceCompletion(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	client := uninstallClientMock{
		listInstances: func(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
			require.True(t, connectivityinternal.IsNonInteractive(ctx))
			require.Equal(t, connectivityclient.ListOptions{PageSize: 100}, options)
			return &connectivityclient.ConnectorInstanceList{Items: []connectivityclient.ConnectorInstance{instance}}, nil
		},
	}
	command := NewUninstallCommand(factoryReturning(client))

	require.Equal(t, "uninstall <connectorinstance>", command.Use)
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
	factory := func(*cobra.Command) (connectivityclient.Client, error) {
		usedFactory = true
		return nil, errors.New("factory must not be used")
	}

	_, err := executeCommand(NewUninstallCommand(factory), "stripe-eu", "extra", "--confirm")

	require.Error(t, err)
	require.False(t, usedFactory)
}
