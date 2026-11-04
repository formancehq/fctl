package connectivity

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/cmd/connectivity/internal/client"
	fctl "github.com/formancehq/fctl/v3/pkg"
	connectivitymodel "github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type statusClientMock struct {
	connectivityclient.Client
	getInfo func(context.Context) (*connectivitymodel.Info, error)
}

func (m statusClientMock) GetInfo(ctx context.Context) (*connectivitymodel.Info, error) {
	return m.getInfo(ctx)
}

type statusContextKey struct{}

func executeStatus(ctx context.Context, factory connectivityinternal.ClientFactory, args ...string) (string, error) {
	command := NewStatusCommand(factory)
	command.Flags().String(fctl.OutputFlag, "plain", "")
	command.SilenceErrors = true
	command.SilenceUsage = true
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	err := command.ExecuteContext(ctx)
	return out.String(), err
}

func TestStatusShowsBuildInformationInPlainAndFlatJSON(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		t.Run(format, func(t *testing.T) {
			calls := 0
			ctx := context.WithValue(t.Context(), statusContextKey{}, "status caller")
			factory := func(cmd *cobra.Command) (connectivityclient.Client, error) {
				require.Same(t, ctx, cmd.Context())
				return statusClientMock{getInfo: func(got context.Context) (*connectivitymodel.Info, error) {
					calls++
					require.Same(t, ctx, got)
					return &connectivitymodel.Info{Version: "v1.2.3", Commit: "abc123", BuildDate: "2026-10-28T22:29:14Z"}, nil
				}}, nil
			}
			out, err := executeStatus(ctx, factory, "--output", format)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			if format == "json" {
				require.JSONEq(t, `{"data":{"version":"v1.2.3","commit":"abc123","buildDate":"2026-10-28T22:29:14Z"}}`, out)
			} else {
				require.Equal(t, "Version: v1.2.3\nCommit: abc123\nBuild Date: 2026-10-28T22:29:14Z\n", out)
			}
		})
	}
}

func TestStatusReturnsFactoryAPINilClientAndEmptyResponseErrors(t *testing.T) {
	apiErr := &connectivityclient.APIError{StatusCode: 401, Message: "Unauthorized"}
	for _, tt := range []struct {
		name    string
		factory connectivityinternal.ClientFactory
		want    string
	}{
		{"missing factory", nil, "factory is required"},
		{"factory", func(*cobra.Command) (connectivityclient.Client, error) {
			return nil, errors.New("authentication failed")
		}, "authentication failed"},
		{"missing client", func(*cobra.Command) (connectivityclient.Client, error) { return nil, nil }, "client is required"},
		{"API", func(*cobra.Command) (connectivityclient.Client, error) {
			return statusClientMock{getInfo: func(context.Context) (*connectivitymodel.Info, error) { return nil, apiErr }}, nil
		}, "Unauthorized"},
		{"empty response", func(*cobra.Command) (connectivityclient.Client, error) {
			return statusClientMock{getInfo: func(context.Context) (*connectivitymodel.Info, error) { return nil, nil }}, nil
		}, "empty response"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, format := range []string{"plain", "json"} {
				out, err := executeStatus(t.Context(), tt.factory, "--output", format)
				require.ErrorContains(t, err, tt.want)
				require.Empty(t, out)
				if tt.name == "API" {
					require.ErrorIs(t, err, apiErr)
				}
			}
		})
	}
}

func TestStatusRejectsPositionalArgumentsBeforeAuthentication(t *testing.T) {
	called := false
	factory := func(*cobra.Command) (connectivityclient.Client, error) { called = true; return nil, nil }
	out, err := executeStatus(t.Context(), factory, "extra")
	require.Error(t, err)
	require.False(t, called)
	require.Empty(t, out)
}
