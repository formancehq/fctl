package connections

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type mutationClientMock struct {
	connectorVersions
	listConnectors func(context.Context, connectivityinternal.ListOptions) (*testConnectorList, error)
	capabilities   func(context.Context) (*connectivity.QueryCapabilities, error)
	create         func(context.Context, connectivity.ConnectionCreate) (*connectivity.Connection, error)
}

func TestInstallBuildsConnectionFromVersionSchemaAndOnlySetsChangedScalars(t *testing.T) {
	var gotConnector, gotVersion string
	var gotBody connectivity.ConnectionCreate
	client := mutationClientMock{
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, connector, version string) (*connectivity.ConnectorVersion, error) {
				gotConnector, gotVersion = connector, version
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(_ context.Context, body connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			gotBody = body
			return &connectivity.Connection{Name: body.Name, ConnectionConfiguration: createdConfiguration(body.ConnectionConfiguration)}, nil
		},
	}
	read := mockReadFile(map[string]string{
		"install.yaml":  "env:\n  API_URL: https://config.example\n",
		"connector.env": "TIMEOUT=45s\n",
		"token.txt":     "file-token",
	})

	output, err := executeCommand(
		testNewInstallCommand(factoryReturning(client), read, mockPathCompleter(nil)),
		"stripe", "--name=stripe-eu", "--ledger=main", "--version=2.0.0",
		"--poll-interval=3s", "--config=install.yaml",
		"--env-file=connector.env", "--set=TOKEN=@token.txt", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "stripe", gotConnector)
	require.Equal(t, "2.0.0", gotVersion, "the pinned version supplies the config schema")
	require.Equal(t, "stripe-eu", gotBody.Name)
	require.Equal(t, "stripe", gotBody.ConnectionConfiguration.Connector)
	require.Equal(t, "main", gotBody.ConnectionConfiguration.Ledger)
	require.Equal(t, "2.0.0", *gotBody.ConnectionConfiguration.Version)
	require.Equal(t, "3s", *gotBody.ConnectionConfiguration.PollInterval)
	require.Equal(t, "https://config.example", gotBody.ConnectionConfiguration.Env["API_URL"])
	require.Equal(t, "file-token", gotBody.ConnectionConfiguration.Env["TOKEN"])
	require.Equal(t, "45s", gotBody.ConnectionConfiguration.Env["TIMEOUT"])
	require.Empty(t, gotBody.ConnectionConfiguration.Files)
	require.Equal(t, `Connection "stripe-eu" installed with connector "stripe" for ledger "main".`, strings.TrimSpace(output))
}

func TestInstallWithoutSelectorUsesStableHeadForTheSchema(t *testing.T) {
	var gotVersion string
	var gotBody connectivity.ConnectionCreate
	client := mutationClientMock{
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivity.ConnectorVersion, error) {
				gotVersion = version
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(_ context.Context, body connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			gotBody = body
			return &connectivity.Connection{Name: body.Name, ConnectionConfiguration: createdConfiguration(body.ConnectionConfiguration)}, nil
		},
	}

	_, err := executeCommand(
		testNewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe", "--ledger=main", "--set=API_URL=https://example", "--confirm",
	)

	require.NoError(t, err)
	// The default stable schema and the explicit wire selector must agree.
	require.Equal(t, "stable", gotVersion)
	require.Nil(t, gotBody.ConnectionConfiguration.Version)
	require.Equal(t, "stable", string(*gotBody.ConnectionConfiguration.Channel), "the CLI sends its default explicitly")
}

func TestInstallWithChannelFlagTracksTheChannelAndUsesItsHeadForTheSchema(t *testing.T) {
	var gotVersion string
	var gotBody connectivity.ConnectionCreate
	client := mutationClientMock{
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivity.ConnectorVersion, error) {
				gotVersion = version
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(_ context.Context, body connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			gotBody = body
			return &connectivity.Connection{Name: body.Name, ConnectionConfiguration: createdConfiguration(body.ConnectionConfiguration)}, nil
		},
	}

	_, err := executeCommand(
		testNewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe", "--ledger=main", "--channel=stable", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "stable", gotVersion, "the channel alias resolves the schema exactly as installation will")
	require.NotNil(t, gotBody.ConnectionConfiguration.Channel)
	require.Equal(t, "stable", string(*gotBody.ConnectionConfiguration.Channel))
	require.Nil(t, gotBody.ConnectionConfiguration.Version)
}

func TestInstallPreservesNativeMissingStableVersionError(t *testing.T) {
	apiError := &connectivity.Error{ErrorCode: fctl.Ptr("channel_empty"), ErrorMessage: fctl.Ptr("only v2.0.0-beta.1 is published")}
	client := mutationClientMock{connectorVersions: connectorVersions{
		getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
			return nil, apiError
		},
	}}

	_, err := executeCommand(testNewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe", "--ledger=main", "--confirm")

	require.ErrorIs(t, err, apiError)
	var nativeError *connectivity.Error
	require.ErrorAs(t, err, &nativeError)
	require.Same(t, apiError, nativeError)
	require.Equal(t, "only v2.0.0-beta.1 is published", *nativeError.ErrorMessage)
}

func TestInstallDefaultsNameToConnectorAndOmitsUnsetOptionalData(t *testing.T) {
	var gotBody connectivity.ConnectionCreate
	client := mutationClientMock{
		create: func(_ context.Context, body connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			gotBody = body
			return &connectivity.Connection{Name: body.Name, ConnectionConfiguration: createdConfiguration(body.ConnectionConfiguration)}, nil
		},
	}

	_, err := executeCommand(testNewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "wise", "--ledger=main", "--confirm")

	require.NoError(t, err)
	require.Equal(t, "wise", gotBody.Name)
	require.Nil(t, gotBody.ConnectionConfiguration.Version)
	require.Equal(t, "stable", string(*gotBody.ConnectionConfiguration.Channel))
	require.Nil(t, gotBody.ConnectionConfiguration.InitialCursor)
	require.Nil(t, gotBody.ConnectionConfiguration.PollInterval)
	require.Empty(t, gotBody.ConnectionConfiguration.Env)
	require.Empty(t, gotBody.ConnectionConfiguration.Files)
}

func TestInstallRequiresLedgerBeforeUsingClient(t *testing.T) {
	usedClient := false
	factory := func(*cobra.Command) (*sdk.FormanceV1, error) {
		usedClient = true
		return nil, errors.New("client must not be used")
	}

	_, err := executeCommand(testNewInstallCommand(factory, mockReadFile(nil), mockPathCompleter(nil)), "stripe", "--confirm")

	require.ErrorContains(t, err, `required flag(s) "ledger" not set`)
	require.False(t, usedClient)
}

func TestInstallConfirmationRejectionPreventsCreate(t *testing.T) {
	created := false
	client := mutationClientMock{
		create: func(context.Context, connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			created = true
			return nil, errors.New("CreateConnection must not be called")
		},
	}
	controller := NewInstallController(mockReadFile(nil))
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }
	command := testNewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	require.NoError(t, command.ParseFlags([]string{"--ledger=main"}))

	command.SetContext(context.Background())
	_, err := controller.run(command, []string{"stripe"}, mockSDK(client))

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.False(t, created)
}

func TestInstallValidationErrorPreventsCreate(t *testing.T) {
	created := false
	client := mutationClientMock{
		connectorVersions: connectorVersions{
			getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(context.Context, connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			created = true
			return nil, errors.New("CreateConnection must not be called")
		},
	}

	_, err := executeCommand(
		testNewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe", "--ledger=main", "--set=UNKNOWN=value", "--confirm",
	)

	require.ErrorContains(t, err, "unknown configuration keys: UNKNOWN")
	require.False(t, created)
}

func TestInstallPreservesVersionAndCreateAPIErrorsAndRejectsEmptyResponses(t *testing.T) {
	versionError := &connectivity.Error{ErrorCode: fctl.Ptr("CONNECTOR_NOT_FOUND"), ErrorMessage: fctl.Ptr("missing")}
	createError := &connectivity.Error{ErrorCode: fctl.Ptr("CONNECTORINSTANCE_EXISTS"), ErrorMessage: fctl.Ptr("duplicate")}
	tests := map[string]struct {
		client mutationClientMock
		want   error
		text   string
	}{
		"version API": {
			client: mutationClientMock{connectorVersions: connectorVersions{
				getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
					return nil, versionError
				},
			}},
			want: versionError,
		},
		"empty version": {
			client: mutationClientMock{connectorVersions: connectorVersions{
				getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) { return nil, nil },
			}},
			text: "empty response",
		},
		"create API": {
			client: mutationClientMock{
				create: func(context.Context, connectivity.ConnectionCreate) (*connectivity.Connection, error) {
					return nil, createError
				},
			},
			want: createError,
		},
		"empty create": {
			client: mutationClientMock{
				create: func(context.Context, connectivity.ConnectionCreate) (*connectivity.Connection, error) {
					return nil, nil
				},
			},
			text: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(testNewInstallCommand(factoryReturning(test.client), mockReadFile(nil), mockPathCompleter(nil)), "stripe", "--ledger=main", "--confirm")
			require.Error(t, err)
			if test.want != nil {
				require.ErrorIs(t, err, test.want)
			}
			if test.text != "" {
				require.ErrorContains(t, err, test.text)
			}
		})
	}
}

func TestInstallJSONStoresCompleteReturnedConnection(t *testing.T) {
	returned := instanceFixture("stripe-eu")
	setInstanceConfig(&returned, &connectivity.ConnectionConfiguration{Env: map[string]string{
		"TOKEN": "preserved-in-json",
	}})
	client := mutationClientMock{
		create: func(context.Context, connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			return &returned, nil
		},
	}
	command := testNewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "stripe", "--ledger=main", "--confirm", "--output=json")

	require.NoError(t, err)
	var envelope struct {
		Data InstallStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, returned, envelope.Data.Connection)
}

func TestInstallCompletionReturnsVersionsSchemaKeysAndFileCandidates(t *testing.T) {
	connector := connectivity.Connector{
		Name:        "stripe",
		Description: stringPtr("Stripe ingestion"),
	}
	client := mutationClientMock{
		listConnectors: func(_ context.Context, options connectivityinternal.ListOptions) (*testConnectorList, error) {
			require.Equal(t, connectivityinternal.ListOptions{PageSize: 100}, options)
			return &testConnectorList{Items: []connectivity.Connector{connector}}, nil
		},
		connectorVersions: connectorVersions{
			listVersions: func(ctx context.Context, name string, options connectivityinternal.ListOptions) (*testConnectorVersionList, error) {
				require.True(t, hasTwoSecondDeadline(ctx))
				require.Equal(t, "stripe", name)
				require.Equal(t, int32(100), options.PageSize)
				return versionListFixture(), nil
			},
			getVersion: func(ctx context.Context, name, _ string) (*connectivity.ConnectorVersion, error) {
				require.True(t, hasTwoSecondDeadline(ctx))
				require.Equal(t, "stripe", name)
				return versionWithFullSchema(), nil
			},
		},
	}
	var pathPrefix string
	paths := func(prefix string) ([]string, error) {
		pathPrefix = prefix
		return []string{"fixtures/token.txt"}, nil
	}
	factory := func(cmd *cobra.Command) (*sdk.FormanceV1, error) {
		require.True(t, hasTwoSecondDeadline(cmd.Context()))
		return mockSDK(client), nil
	}
	command := testNewInstallCommand(factory, mockReadFile(nil), paths)

	versionCompletion := testCompleteVersions(factory, installConnectorArgument)
	versions, directive := versionCompletion(command, []string{"stripe"}, "2")
	require.Equal(t, []string{"2.0.0\tlatest"}, versions)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder, directive)

	setCompletion := testCompleteSetValues(factory, resolveInstallConnectorVersion, paths)
	sets, directive := setCompletion(command, []string{"stripe"}, "API")
	require.Equal(t, []string{"API_URL=\trequired · API endpoint"}, sets)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)

	sets, directive = setCompletion(command, []string{"stripe"}, "TOKEN=@fixtures/t")
	require.Equal(t, "fixtures/t", pathPrefix)
	require.Equal(t, []string{"TOKEN=@fixtures/token.txt"}, sets)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}
