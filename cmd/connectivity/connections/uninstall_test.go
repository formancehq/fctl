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
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

type uninstallClientMock struct {
	listInstances func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error)
	delete        func(context.Context, string) error
}

func TestUninstallDeletesConfirmedConnectionAndRendersSuccess(t *testing.T) {
	var deletedName string
	client := uninstallClientMock{delete: func(_ context.Context, name string) error {
		deletedName = name
		return nil
	}}

	output, err := executeCommand(testNewUninstallCommand(factoryReturning(client)), "stripe-eu", "--confirm")

	require.NoError(t, err)
	require.Equal(t, "stripe-eu", deletedName)
	require.Equal(t, `Connection "stripe-eu" uninstall requested.`, strings.TrimSpace(output))
}

func TestUninstallConfirmationRejectionPreventsDelete(t *testing.T) {
	deleted := false
	client := uninstallClientMock{delete: func(context.Context, string) error {
		deleted = true
		return errors.New("DeleteConnection must not be called")
	}}
	controller := NewUninstallController()
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }

	_, err := controller.run(&cobra.Command{}, []string{"stripe-eu"}, mockSDK(client))

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.False(t, deleted)
}

func TestUninstallPreservesDeleteAPIError(t *testing.T) {
	apiError := sdkerrors.NewSDKError("API error occurred", 404, `{"errorCode":"CONNECTORINSTANCE_NOT_FOUND","errorMessage":"missing"}`, nil)
	client := uninstallClientMock{delete: func(context.Context, string) error { return apiError }}

	_, err := executeCommand(testNewUninstallCommand(factoryReturning(client)), "stripe-eu", "--confirm")

	require.ErrorIs(t, err, apiError)
}

func TestUninstallInvalidArgumentsDoNotDelete(t *testing.T) {
	usedFactory := false
	factory := func(*cobra.Command) (*sdk.FormanceV1, error) {
		usedFactory = true
		return nil, errors.New("factory must not be used")
	}

	_, err := executeCommand(testNewUninstallCommand(factory), "stripe-eu", "extra", "--confirm")

	require.Error(t, err)
	require.False(t, usedFactory)
}
