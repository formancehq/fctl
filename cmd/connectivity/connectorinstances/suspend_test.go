package connectorinstances

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type suspensionClientMock struct {
	connectivityclient.Client
	listInstances func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error)
	patch         func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error)
}

func (m suspensionClientMock) ListConnectorInstances(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
	return m.listInstances(ctx, options)
}

func (m suspensionClientMock) PatchConnectorInstance(ctx context.Context, name string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
	return m.patch(ctx, name, patch)
}

func TestSuspensionCommandsPatchOnlyDesiredBooleanAndTreatAlreadyConvergedResponseAsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		command    func(connectivityinternal.ClientFactory) *cobra.Command
		suspend    bool
		wantOutput string
	}{
		{name: "suspend", command: NewSuspendCommand, suspend: true, wantOutput: `Connector instance "stripe-eu" suspension requested.`},
		{name: "unsuspend", command: NewUnsuspendCommand, suspend: false, wantOutput: `Connector instance "stripe-eu" resumption requested.`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := suspensionClientMock{patch: func(_ context.Context, name string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
				calls++
				require.Equal(t, "stripe-eu", name)
				require.Equal(t, connectivityclient.ConnectorInstancePatch{"spec": map[string]any{"suspend": test.suspend}}, patch)
				instance := instanceFixture(name)
				instance.Spec.Suspend = fctl.Ptr(test.suspend)
				return &instance, nil
			}}

			output, err := executeCommand(test.command(factoryReturning(client)), "stripe-eu", "--confirm")

			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.Equal(t, test.wantOutput, strings.TrimSpace(output))
		})
	}
}

func TestSuspensionCommandConfirmationRejectionMakesNoPatch(t *testing.T) {
	patchCalls := 0
	client := suspensionClientMock{patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
		patchCalls++
		return nil, errors.New("PatchConnectorInstance must not be called")
	}}
	controller := NewSuspensionController(factoryReturning(client), true)
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }

	_, err := controller.Run(&cobra.Command{}, []string{"stripe-eu"})

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.Zero(t, patchCalls)
}

func TestSuspensionCommandsPreserveAPIError(t *testing.T) {
	apiError := &connectivityclient.APIError{StatusCode: 409, Code: "CONFLICT", Message: "stale"}
	client := suspensionClientMock{patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
		return nil, apiError
	}}

	for _, command := range []func(connectivityinternal.ClientFactory) *cobra.Command{NewSuspendCommand, NewUnsuspendCommand} {
		_, err := executeCommand(command(factoryReturning(client)), "stripe-eu", "--confirm")
		require.ErrorIs(t, err, apiError)
	}
}

func TestSuspensionCommandsRejectInvalidArgumentsBeforeUsingClient(t *testing.T) {
	usedFactory := false
	factory := func(*cobra.Command) (connectivityclient.Client, error) {
		usedFactory = true
		return nil, errors.New("factory must not be used")
	}

	for _, command := range []func(connectivityinternal.ClientFactory) *cobra.Command{NewSuspendCommand, NewUnsuspendCommand} {
		_, err := executeCommand(command(factory), "stripe-eu", "extra", "--confirm")
		require.Error(t, err)
	}
	require.False(t, usedFactory)
}

func TestSuspensionCommandsRegisterConfirmationCompletionAndRootChildren(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	client := suspensionClientMock{listInstances: func(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
		require.True(t, connectivityinternal.IsNonInteractive(ctx))
		require.Equal(t, connectivityclient.ListOptions{PageSize: 100}, options)
		return &connectivityclient.ConnectorInstanceList{Items: []connectivityclient.ConnectorInstance{instance}}, nil
	}}

	for _, test := range []struct {
		name    string
		command func(connectivityinternal.ClientFactory) *cobra.Command
	}{
		{name: "suspend", command: NewSuspendCommand},
		{name: "unsuspend", command: NewUnsuspendCommand},
	} {
		command := test.command(factoryReturning(client))
		require.Equal(t, test.name+" <connectorinstance>", command.Use)
		require.NotNil(t, command.Flags().Lookup("confirm"))
		require.Error(t, command.Args(command, nil))
		require.NoError(t, command.Args(command, []string{"stripe-eu"}))
		require.Error(t, command.Args(command, []string{"stripe-eu", "extra"}))
		candidates, directive := command.ValidArgsFunction(command, nil, "stripe")
		require.Equal(t, []string{"stripe-eu\tstripe · main"}, candidates)
		require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	}

	root := NewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	for _, name := range []string{"suspend", "unsuspend"} {
		child, _, err := root.Find([]string{name})
		require.NoError(t, err)
		require.Equal(t, name, child.Name())
	}
}
