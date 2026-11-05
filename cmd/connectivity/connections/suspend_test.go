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
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

type suspensionClientMock struct {
	listInstances func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error)
	patch         func(context.Context, string, map[string]any) (*connectivity.Connection, error)
}

func TestSuspensionCommandsPatchOnlyDesiredBooleanAndTreatAlreadyConvergedResponseAsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		command    func(testSDKFactory) *cobra.Command
		suspend    bool
		wantOutput string
	}{
		{name: "suspend", command: testNewSuspendCommand, suspend: true, wantOutput: `Connection "stripe-eu" suspension requested.`},
		{name: "unsuspend", command: testNewUnsuspendCommand, suspend: false, wantOutput: `Connection "stripe-eu" resumption requested.`},
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
	controller := NewSuspensionController(true)
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }

	_, err := controller.run(&cobra.Command{}, []string{"stripe-eu"}, mockSDK(client))

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.Zero(t, patchCalls)
}

func TestSuspensionCommandsPreserveAPIError(t *testing.T) {
	apiError := sdkerrors.NewSDKError("API error occurred", 409, `{"errorCode":"CONFLICT","errorMessage":"stale"}`, nil)
	client := suspensionClientMock{patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
		return nil, apiError
	}}

	for _, command := range []func(testSDKFactory) *cobra.Command{testNewSuspendCommand, testNewUnsuspendCommand} {
		_, err := executeCommand(command(factoryReturning(client)), "stripe-eu", "--confirm")
		require.ErrorIs(t, err, apiError)
	}
}

func TestSuspensionCommandsRejectInvalidArgumentsBeforeUsingClient(t *testing.T) {
	usedFactory := false
	factory := func(*cobra.Command) (*sdk.FormanceV1, error) {
		usedFactory = true
		return nil, errors.New("factory must not be used")
	}

	for _, command := range []func(testSDKFactory) *cobra.Command{testNewSuspendCommand, testNewUnsuspendCommand} {
		_, err := executeCommand(command(factory), "stripe-eu", "extra", "--confirm")
		require.Error(t, err)
	}
	require.False(t, usedFactory)
}

func TestSuspensionCommandsRegisterConfirmationCompletionAndRootChildren(t *testing.T) {
	instance := instanceFixture("stripe-eu")
	client := suspensionClientMock{listInstances: func(ctx context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
		require.True(t, hasTwoSecondDeadline(ctx))
		require.Equal(t, connectivityinternal.ListOptions{PageSize: 100}, options)
		return &testConnectionList{Items: []connectivity.Connection{instance}}, nil
	}}

	for _, test := range []struct {
		name    string
		command func(testSDKFactory) *cobra.Command
	}{
		{name: "suspend", command: testNewSuspendCommand},
		{name: "unsuspend", command: testNewUnsuspendCommand},
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

	root := testNewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	for _, name := range []string{"suspend", "unsuspend"} {
		child, _, err := root.Find([]string{name})
		require.NoError(t, err)
		require.Equal(t, name, child.Name())
	}
}
