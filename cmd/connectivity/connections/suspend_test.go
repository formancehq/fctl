package connections

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type suspensionClientMock struct {
	connectivityinternal.API
	listInstances func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error)
	patch         func(context.Context, string, map[string]any) (*connectivity.Connection, error)
}

func (m suspensionClientMock) ListConnections(ctx context.Context, request operations.ListConnectionsRequest, security *operations.ListConnectionsSecurity, opts ...operations.Option) (*operations.ListConnectionsResponse, error) {
	data, err := m.listInstances(ctx, listOptions(request.PageSize, request.Cursor, request.Query))
	response := &operations.ListConnectionsResponse{StatusCode: 200}
	if data != nil {
		response.ConnectionCursor = &connectivity.ConnectionCursor{Cursor: connectivity.Cursor{Data: data.Items, HasMore: data.HasMore, Next: connectivityinternal.OptionalString(data.Next), PageSize: int(data.PageSize)}}
	}
	return response, err
}

func (m suspensionClientMock) PatchConnection(ctx context.Context, request operations.PatchConnectionRequest, security *operations.PatchConnectionSecurity, opts ...operations.Option) (*operations.PatchConnectionResponse, error) {
	patch, err := sdkPatchBody(ctx, request)
	if err != nil {
		return nil, err
	}
	data, err := m.patch(ctx, request.Name, patch)
	return &operations.PatchConnectionResponse{StatusCode: 200, Connection: data}, err
}

func TestSuspensionCommandsPatchOnlyDesiredBooleanAndTreatAlreadyConvergedResponseAsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		command    func(connectivityinternal.ClientFactory) *cobra.Command
		suspend    bool
		wantOutput string
	}{
		{name: "suspend", command: NewSuspendCommand, suspend: true, wantOutput: `Connection "stripe-eu" suspension requested.`},
		{name: "unsuspend", command: NewUnsuspendCommand, suspend: false, wantOutput: `Connection "stripe-eu" resumption requested.`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := suspensionClientMock{patch: func(_ context.Context, name string, patch map[string]any) (*connectivity.Connection, error) {
				calls++
				require.Equal(t, "stripe-eu", name)
				require.Equal(t, map[string]any{"configuration": map[string]any{"suspend": test.suspend}}, patch)
				instance := instanceFixture(name)
				instance.ConnectionConfiguration.Suspend = fctl.Ptr(test.suspend)
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
	client := suspensionClientMock{patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
		patchCalls++
		return nil, errors.New("PatchConnection must not be called")
	}}
	controller := NewSuspensionController(factoryReturning(client), true)
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }

	_, err := controller.Run(&cobra.Command{}, []string{"stripe-eu"})

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.Zero(t, patchCalls)
}

func TestSuspensionCommandsPreserveAPIError(t *testing.T) {
	apiError := &connectivityinternal.APIError{StatusCode: 409, Code: "CONFLICT", Message: "stale"}
	client := suspensionClientMock{patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
		return nil, apiError
	}}

	for _, command := range []func(connectivityinternal.ClientFactory) *cobra.Command{NewSuspendCommand, NewUnsuspendCommand} {
		_, err := executeCommand(command(factoryReturning(client)), "stripe-eu", "--confirm")
		require.ErrorIs(t, err, apiError)
	}
}

func TestSuspensionCommandsRejectInvalidArgumentsBeforeUsingClient(t *testing.T) {
	usedFactory := false
	factory := func(*cobra.Command) (connectivityinternal.API, error) {
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
	client := suspensionClientMock{listInstances: func(ctx context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
		require.True(t, connectivityinternal.IsNonInteractive(ctx))
		require.Equal(t, connectivityinternal.ListOptions{PageSize: 100}, options)
		return &testConnectionList{Items: []connectivity.Connection{instance}}, nil
	}}

	for _, test := range []struct {
		name    string
		command func(connectivityinternal.ClientFactory) *cobra.Command
	}{
		{name: "suspend", command: NewSuspendCommand},
		{name: "unsuspend", command: NewUnsuspendCommand},
	} {
		command := test.command(factoryReturning(client))
		require.Equal(t, test.name+" <connection>", command.Use)
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
