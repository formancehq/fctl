package connections

import (
	"context"
	"encoding/json"
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

type mutationClientMock struct {
	connectorVersions
	listConnectors func(context.Context, connectivityinternal.ListOptions) (*testConnectorList, error)
	capabilities   func(context.Context) (*connectivity.QueryCapabilities, error)
	create         func(context.Context, connectivity.ConnectionCreate) (*connectivity.Connection, error)
}

func (m mutationClientMock) ListConnectors(ctx context.Context, request operations.ListConnectorsRequest, security *operations.ListConnectorsSecurity, opts ...operations.Option) (*operations.ListConnectorsResponse, error) {
	data, err := m.listConnectors(ctx, listOptions(request.PageSize, request.Cursor, request.Query))
	response := &operations.ListConnectorsResponse{StatusCode: 200}
	if data != nil {
		response.ConnectorCursor = &connectivity.ConnectorCursor{Cursor: connectivity.ConnectorCursorCursor{Data: data.Items, HasMore: data.HasMore, Next: connectivityinternal.OptionalString(data.Next), PageSize: int(data.PageSize)}}
	}
	return response, err
}

func (m mutationClientMock) GetQueryCapabilities(ctx context.Context, security *operations.GetQueryCapabilitiesSecurity, opts ...operations.Option) (*operations.GetQueryCapabilitiesResponse, error) {
	data, err := m.capabilities(ctx)
	return &operations.GetQueryCapabilitiesResponse{StatusCode: 200, QueryCapabilities: data}, err
}

func (m mutationClientMock) CreateConnection(ctx context.Context, body connectivity.ConnectionCreate, security *operations.CreateConnectionSecurity, opts ...operations.Option) (*operations.CreateConnectionResponse, error) {
	data, err := m.create(ctx, body)
	return &operations.CreateConnectionResponse{StatusCode: 201, Connection: data}, err
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
		NewInstallCommand(factoryReturning(client), read, mockPathCompleter(nil)),
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
		NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
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
		NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe", "--ledger=main", "--channel=stable", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "stable", gotVersion, "the channel alias resolves the schema exactly as installation will")
	require.NotNil(t, gotBody.ConnectionConfiguration.Channel)
	require.Equal(t, "stable", string(*gotBody.ConnectionConfiguration.Channel))
	require.Nil(t, gotBody.ConnectionConfiguration.Version)
}

func TestInstallReportsPrereleaseOnlyCataloguesAsMissingStableVersion(t *testing.T) {
	client := mutationClientMock{connectorVersions: connectorVersions{
		getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
			return nil, &connectivityinternal.APIError{StatusCode: 404, Code: "channel_empty", Message: "only v2.0.0-beta.1 is published"}
		},
	}}

	_, err := executeCommand(NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe", "--ledger=main", "--confirm")

	require.ErrorContains(t, err, `connectivity connector "stripe" has no published version`)
}

func TestInstallDefaultsNameToConnectorAndOmitsUnsetOptionalData(t *testing.T) {
	var gotBody connectivity.ConnectionCreate
	client := mutationClientMock{
		create: func(_ context.Context, body connectivity.ConnectionCreate) (*connectivity.Connection, error) {
			gotBody = body
			return &connectivity.Connection{Name: body.Name, ConnectionConfiguration: createdConfiguration(body.ConnectionConfiguration)}, nil
		},
	}

	_, err := executeCommand(NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "wise", "--ledger=main", "--confirm")

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
	factory := func(*cobra.Command) (connectivityinternal.API, error) {
		usedClient = true
		return nil, errors.New("client must not be used")
	}

	_, err := executeCommand(NewInstallCommand(factory, mockReadFile(nil), mockPathCompleter(nil)), "stripe", "--confirm")

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
	controller := NewInstallController(factoryReturning(client), mockReadFile(nil))
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }
	command := NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	require.NoError(t, command.ParseFlags([]string{"--ledger=main"}))

	_, err := controller.Run(command, []string{"stripe"})

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
		NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe", "--ledger=main", "--set=UNKNOWN=value", "--confirm",
	)

	require.ErrorContains(t, err, "unknown configuration keys: UNKNOWN")
	require.False(t, created)
}

func TestInstallPreservesVersionAndCreateAPIErrorsAndRejectsEmptyResponses(t *testing.T) {
	versionError := &connectivityinternal.APIError{StatusCode: 404, Code: "CONNECTOR_NOT_FOUND", Message: "missing"}
	createError := &connectivityinternal.APIError{StatusCode: 409, Code: "CONNECTORINSTANCE_EXISTS", Message: "duplicate"}
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
			_, err := executeCommand(NewInstallCommand(factoryReturning(test.client), mockReadFile(nil), mockPathCompleter(nil)), "stripe", "--ledger=main", "--confirm")
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
	command := NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "stripe", "--ledger=main", "--confirm", "--output=json")

	require.NoError(t, err)
	var envelope struct {
		Data InstallStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, returned, envelope.Data.Connection)
}

func TestInstallRegistersAliasesAndRootIntegration(t *testing.T) {
	standalone := NewInstallCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	require.Equal(t, "install <connector>", standalone.Use)
	require.Equal(t, []string{"create", "in"}, standalone.Aliases)
	require.Error(t, standalone.Args(standalone, nil))
	require.NoError(t, standalone.Args(standalone, []string{"stripe"}))
	require.Error(t, standalone.Args(standalone, []string{"stripe", "extra"}))

	root := NewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	child, _, err := root.Find([]string{"install"})
	require.NoError(t, err)
	require.Equal(t, "install", child.Name())
	require.True(t, reflect.DeepEqual([]string{"create", "in"}, child.Aliases))
}

func TestInstallWiresChannelCompletionFromCapabilities(t *testing.T) {
	client := mutationClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
		return &connectivity.QueryCapabilities{Resources: map[string]map[string]connectivity.QueryFieldCapability{
			connectivityinternal.ResourceConnections: {
				"channel": {Operators: []string{"$match"}, Enum: []string{"stable", "rc", "beta", "alpha"}},
			},
		}}, nil
	}}
	command := NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))

	channelCompletion, ok := command.GetFlagCompletionFunc("channel")
	require.True(t, ok)
	channels, directive := channelCompletion(command, nil, "s")
	require.Equal(t, []string{"stable"}, channels)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestInstallDoesNotExposeUnsupportedStartSequence(t *testing.T) {
	command := NewInstallCommand(nil, mockReadFile(nil), mockPathCompleter(nil))

	require.Nil(t, command.Flags().Lookup("start-sequence"))
}

func TestInstallWiresConnectorVersionSetAndFileCompletions(t *testing.T) {
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
				require.True(t, connectivityinternal.IsNonInteractive(ctx))
				require.Equal(t, "stripe", name)
				require.Equal(t, int32(100), options.PageSize)
				return versionListFixture(), nil
			},
			getVersion: func(ctx context.Context, name, _ string) (*connectivity.ConnectorVersion, error) {
				require.True(t, connectivityinternal.IsNonInteractive(ctx))
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
	factory := func(cmd *cobra.Command) (connectivityinternal.API, error) {
		require.True(t, connectivityinternal.IsNonInteractive(cmd.Context()))
		return client, nil
	}
	command := NewInstallCommand(factory, mockReadFile(nil), paths)

	connectors, directive := command.ValidArgsFunction(command, nil, "str")
	require.Equal(t, []string{"stripe\tStripe ingestion"}, connectors)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	versionCompletion, ok := command.GetFlagCompletionFunc("version")
	require.True(t, ok)
	versions, directive := versionCompletion(command, []string{"stripe"}, "2")
	require.Equal(t, []string{"2.0.0"}, versions)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	setCompletion, ok := command.GetFlagCompletionFunc("set")
	require.True(t, ok)
	sets, directive := setCompletion(command, []string{"stripe"}, "API")
	require.Equal(t, []string{"API_URL=\tAPI endpoint"}, sets)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)

	sets, directive = setCompletion(command, []string{"stripe"}, "TOKEN=@fixtures/t")
	require.Equal(t, "fixtures/t", pathPrefix)
	require.Equal(t, []string{"TOKEN=@fixtures/token.txt"}, sets)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	_, configRegistered := command.GetFlagCompletionFunc("config")
	_, envFileRegistered := command.GetFlagCompletionFunc("env-file")
	require.False(t, configRegistered, "--config must retain Cobra's normal file completion")
	require.False(t, envFileRegistered, "--env-file must retain Cobra's normal file completion")
}
