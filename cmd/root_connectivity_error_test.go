package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	connectivitymodels "github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func TestConnectivityErrorPartsGivesEmptyErrorsAnExplicitMessage(t *testing.T) {
	code, message := connectivityErrorParts(&connectivitymodels.Error{})
	require.Equal(t, "UNKNOWN", code)
	require.NotEmpty(t, message)

	code, message = connectivityErrorParts(&connectivitymodels.Error{ErrorCode: fctl.Ptr("not_found"), ErrorMessage: fctl.Ptr("connector missing")})
	require.Equal(t, "not_found", code)
	require.Equal(t, "connector missing", message)
}
