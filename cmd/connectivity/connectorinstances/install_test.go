package connectorinstances

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
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type mutationClientMock struct {
	connectorVersions
	listConnectors func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error)
	capabilities   func(context.Context) (*connectivityclient.QueryCapabilities, error)
	create         func(context.Context, connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error)
}

func (m mutationClientMock) ListConnectors(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
	return m.listConnectors(ctx, options)
}

func (m mutationClientMock) GetQueryCapabilities(ctx context.Context) (*connectivityclient.QueryCapabilities, error) {
	return m.capabilities(ctx)
}

func (m mutationClientMock) CreateConnectorInstance(ctx context.Context, body connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
	return m.create(ctx, body)
}

func TestInstallBuildsConnectorInstanceFromVersionSchemaAndOnlySetsChangedScalars(t *testing.T) {
	var gotConnector, gotVersion string
	var gotBody connectivityclient.ConnectorInstanceCreate
	client := mutationClientMock{
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
				gotConnector, gotVersion = connector, version
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(_ context.Context, body connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
			gotBody = body
			return &connectivityclient.ConnectorInstance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(body.Name)}, Spec: body.Spec}, nil
		},
	}
	read := mockReadFile(map[string]string{
		"install.yaml":  "env:\n  API_URL:\n    value: https://config.example\n",
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
	require.Equal(t, "stripe", gotBody.Spec.Connector)
	require.Equal(t, "main", gotBody.Spec.Ledger)
	require.Equal(t, "2.0.0", *gotBody.Spec.Version)
	require.Equal(t, "3s", *gotBody.Spec.PollInterval)
	require.Equal(t, "https://config.example", *gotBody.Spec.Config.Env["API_URL"].Value)
	require.Equal(t, "file-token", *gotBody.Spec.Config.Env["TOKEN"].Value)
	require.Equal(t, "45s", *gotBody.Spec.Config.Env["TIMEOUT"].Value)
	require.Empty(t, gotBody.Spec.Config.Files)
	require.Equal(t, `Connector instance "stripe-eu" installed with connector "stripe" for ledger "main".`, strings.TrimSpace(output))
}

func TestInstallWithoutSelectorUsesStableHeadForTheSchema(t *testing.T) {
	var gotVersion string
	var gotBody connectivityclient.ConnectorInstanceCreate
	client := mutationClientMock{
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
				gotVersion = version
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(_ context.Context, body connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
			gotBody = body
			return &connectivityclient.ConnectorInstance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(body.Name)}, Spec: body.Spec}, nil
		},
	}

	_, err := executeCommand(
		NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe", "--ledger=main", "--set=API_URL=https://example", "--confirm",
	)

	require.NoError(t, err)
	// A newer prerelease must not supply the schema for a selector-less
	// install: the API persists the stable head in that case.
	require.Equal(t, "stable", gotVersion)
	require.Nil(t, gotBody.Spec.Version, "an unpinned install must let the server resolve the version")
}

func TestInstallWithChannelFlagTracksTheChannelAndUsesItsHeadForTheSchema(t *testing.T) {
	var gotVersion string
	var gotBody connectivityclient.ConnectorInstanceCreate
	client := mutationClientMock{
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
				gotVersion = version
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(_ context.Context, body connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
			gotBody = body
			return &connectivityclient.ConnectorInstance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(body.Name)}, Spec: body.Spec}, nil
		},
	}

	_, err := executeCommand(
		NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe", "--ledger=main", "--channel=stable", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "stable", gotVersion, "the channel alias resolves the schema exactly as installation will")
	require.NotNil(t, gotBody.Spec.Channel)
	require.Equal(t, "stable", *gotBody.Spec.Channel)
	require.Nil(t, gotBody.Spec.Version)
}

func TestInstallReportsPrereleaseOnlyCataloguesAsMissingStableVersion(t *testing.T) {
	client := mutationClientMock{connectorVersions: connectorVersions{
		getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
			return nil, &connectivityclient.APIError{StatusCode: 404, Code: "channel_empty", Message: "only v2.0.0-beta.1 is published"}
		},
	}}

	_, err := executeCommand(NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe", "--ledger=main", "--confirm")

	require.ErrorContains(t, err, `connectivity connector "stripe" has no published version`)
}

func TestInstallDefaultsNameToConnectorAndOmitsUnsetOptionalData(t *testing.T) {
	var gotBody connectivityclient.ConnectorInstanceCreate
	client := mutationClientMock{
		create: func(_ context.Context, body connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
			gotBody = body
			return &connectivityclient.ConnectorInstance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(body.Name)}, Spec: body.Spec}, nil
		},
	}

	_, err := executeCommand(NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "wise", "--ledger=main", "--confirm")

	require.NoError(t, err)
	require.Equal(t, "wise", gotBody.Name)
	require.Nil(t, gotBody.Spec.Version)
	require.Nil(t, gotBody.Spec.Channel)
	require.Nil(t, gotBody.Spec.StartSequence)
	require.Nil(t, gotBody.Spec.PollInterval)
	require.Nil(t, gotBody.Spec.Config)
}

func TestInstallRequiresLedgerBeforeUsingClient(t *testing.T) {
	usedClient := false
	factory := func(*cobra.Command) (connectivityclient.Client, error) {
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
		create: func(context.Context, connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
			created = true
			return nil, errors.New("CreateConnectorInstance must not be called")
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
			getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
				return versionWithOptionalSchema(), nil
			},
		},
		create: func(context.Context, connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
			created = true
			return nil, errors.New("CreateConnectorInstance must not be called")
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
	versionError := &connectivityclient.APIError{StatusCode: 404, Code: "CONNECTOR_NOT_FOUND", Message: "missing"}
	createError := &connectivityclient.APIError{StatusCode: 409, Code: "CONNECTORINSTANCE_EXISTS", Message: "duplicate"}
	tests := map[string]struct {
		client mutationClientMock
		want   error
		text   string
	}{
		"version API": {
			client: mutationClientMock{connectorVersions: connectorVersions{
				getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
					return nil, versionError
				},
			}},
			want: versionError,
		},
		"empty version": {
			client: mutationClientMock{connectorVersions: connectorVersions{
				getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) { return nil, nil },
			}},
			text: "empty response",
		},
		"create API": {
			client: mutationClientMock{
				create: func(context.Context, connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
					return nil, createError
				},
			},
			want: createError,
		},
		"empty create": {
			client: mutationClientMock{
				create: func(context.Context, connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
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

func TestInstallJSONStoresCompleteReturnedConnectorInstance(t *testing.T) {
	returned := instanceFixture("stripe-eu")
	returned.Metadata.Labels = map[string]string{"region": "eu"}
	returned.Spec.Config = &connectivityclient.ConnectorInstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"TOKEN": {Value: stringPtr("preserved-in-json")},
	}}
	client := mutationClientMock{
		create: func(context.Context, connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
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
	require.Equal(t, returned, envelope.Data.ConnectorInstance)
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
	client := mutationClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
		return &connectivityclient.QueryCapabilities{Resources: map[string]map[string]connectivityclient.QueryFieldCapability{
			connectivityclient.ResourceConnectorInstances: {
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
	connector := connectivityclient.Connector{
		Metadata: connectivityclient.ObjectMeta{Name: stringPtr("stripe")},
		Spec:     connectivityclient.ConnectorSpec{Description: stringPtr("Stripe ingestion")},
	}
	client := mutationClientMock{
		listConnectors: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
			require.Equal(t, connectivityclient.ListOptions{PageSize: 100}, options)
			return &connectivityclient.ConnectorList{Items: []connectivityclient.Connector{connector}}, nil
		},
		connectorVersions: connectorVersions{
			listVersions: func(ctx context.Context, name string, options connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error) {
				require.True(t, connectivityinternal.IsNonInteractive(ctx))
				require.Equal(t, "stripe", name)
				require.Equal(t, int32(100), options.PageSize)
				return versionListFixture(), nil
			},
			getVersion: func(ctx context.Context, name, _ string) (*connectivityclient.ConnectorVersion, error) {
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
	factory := func(cmd *cobra.Command) (connectivityclient.Client, error) {
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
	require.Equal(t, []string{"2.0.0\texample/connector:2.0.0"}, versions)
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
