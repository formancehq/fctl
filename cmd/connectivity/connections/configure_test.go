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
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

type configureClientMock struct {
	connectorVersions
	listInstances func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error)
	getInstance   func(context.Context, string) (*connectivity.Connection, error)
	patch         func(context.Context, string, map[string]any) (*connectivity.Connection, error)
}

func TestConfigureBuildsNestedPatchFromCurrentConfigAndChangedScalars(t *testing.T) {
	current := instanceFixture("stripe-eu")
	setInstanceConfig(&current, &connectivity.ConnectionConfiguration{
		Env:   map[string]string{"API_URL": "https://current.example"},
		Files: map[string]string{"/etc/a": "old", "/etc/b": "kept"},
	})
	returned := instanceFixture("stripe-eu")
	var order []string
	var gotPatch map[string]any
	client := configureClientMock{
		getInstance: func(_ context.Context, name string) (*connectivity.Connection, error) {
			order = append(order, "instance:"+name)
			return &current, nil
		},
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, connector, version string) (*connectivity.ConnectorVersion, error) {
				order = append(order, "version:"+connector+"@"+version)
				return versionWithFileSchema(), nil
			},
		},
		patch: func(_ context.Context, name string, patch map[string]any) (*connectivity.Connection, error) {
			order = append(order, "patch:"+name)
			gotPatch = patch
			return &returned, nil
		},
	}

	output, err := executeCommand(
		testNewConfigureCommand(factoryReturning(client), mockReadFile(map[string]string{"new": "changed"}), mockPathCompleter(nil)),
		"stripe-eu", "--set=/etc/a=@new", "--version=3.0.0",
		"--poll-interval=15s", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, []string{"instance:stripe-eu", "version:stripe@3.0.0", "patch:stripe-eu"}, order,
		"the requested version supplies the schema the new configuration is validated against")
	spec, ok := gotPatch["configuration"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "3.0.0", spec["version"])
	require.NotContains(t, spec, "connector")
	require.NotContains(t, spec, "ledger")
	require.Equal(t, "15s", spec["pollInterval"])
	require.Len(t, spec, 4, "bindings and unchanged fields must not be patched")
	require.NotContains(t, spec, "config")
	require.NotContains(t, spec, "env", "untouched environment entries must not be patched")
	require.Contains(t, spec, "channel")
	require.Nil(t, spec["channel"])
	require.Equal(t, map[string]any{"/etc/a": "changed"}, spec["files"], "untouched files must be omitted from merge patch")

	require.Equal(t, `Connection "stripe-eu" configured.`, strings.TrimSpace(output))
}

func TestConfigureWithoutVersionFlagUsesTheInstancePin(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.ConnectionConfiguration.Version = stringPtr("1.4.2")
	setInstanceConfig(&current, &connectivity.ConnectionConfiguration{})
	var gotVersion string
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivity.ConnectorVersion, error) {
				gotVersion = version
				return versionWithFileSchema(), nil
			},
		},
		patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
			return &current, nil
		},
	}

	_, err := executeCommand(
		testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe-eu", "--set=/etc/a=kept", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "1.4.2", gotVersion)
}

func TestConfigureRequiresStoredOrExplicitSelector(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.ConnectionConfiguration.Version = nil
	current.ConnectionConfiguration.Channel = nil
	current.ConnectionStatus.AppliedVersion = stringPtr("1.9.9")
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
		patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
			t.Fatal("missing selector must not patch")
			return nil, nil
		},
	}
	_, err := executeCommand(testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--set=/etc/a=kept", "--confirm")
	require.ErrorContains(t, err, "no version/channel selector")
}

func TestConfigureRequiresUnambiguousStoredSelector(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.ConnectionConfiguration.Channel = ptr(connectivity.Channel("stable"))
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
	}
	_, err := executeCommand(testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--set=/etc/a=kept", "--confirm")
	require.ErrorContains(t, err, "conflicting version/channel selectors")
}

func TestConfigureChannelTracksTheResolvedSameMajorHeadAcrossPages(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.ConnectionConfiguration.Version = stringPtr("v1.0.0")
	current.ConnectionConfiguration.Channel = ptr(connectivity.Channel("stable"))
	current.ConnectionStatus.AppliedVersion = stringPtr("v1.0.0")
	setInstanceConfig(&current, &connectivity.ConnectionConfiguration{})
	var gotVersion string
	var gotPatch map[string]any
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
		connectorVersions: connectorVersions{
			listVersions: func(_ context.Context, _ string, options connectivityinternal.ListOptions) (*testConnectorVersionList, error) {
				switch options.Cursor {
				case "":
					return &testConnectorVersionList{Items: []connectivity.ConnectorVersionSummary{
						{Version: "v1.1.0"},
						{Version: "v1.2.0-beta.1"},
					}, HasMore: true, Next: "second"}, nil
				case "second":
					return &testConnectorVersionList{Items: []connectivity.ConnectorVersionSummary{
						{Version: "v1.3.0"},
						{Version: "v2.0.0"},
					}}, nil
				default:
					t.Fatalf("unexpected version cursor %q", options.Cursor)
					return nil, nil
				}
			},
			getVersion: func(_ context.Context, _, version string) (*connectivity.ConnectorVersion, error) {
				gotVersion = version
				return versionWithFileSchema(), nil
			},
		},
		patch: func(_ context.Context, _ string, patch map[string]any) (*connectivity.Connection, error) {
			gotPatch = patch
			return &current, nil
		},
	}

	_, err := executeCommand(
		testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe-eu", "--channel=stable", "--set=/etc/a=kept", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "v1.3.0", gotVersion, "the server selects the highest stable candidate in the running major")
	spec := gotPatch["configuration"].(map[string]any)
	require.Equal(t, "stable", spec["channel"])
	require.Contains(t, spec, "version", "switching to a channel removes an existing pin")
	require.Nil(t, spec["version"])
}

func TestConfigureChannelUsesAppliedVersionForImmutableConnector(t *testing.T) {
	for _, applied := range []*string{stringPtr("v1.2.0"), nil} {
		current := instanceFixture("stripe-eu")
		current.ConnectionConfiguration.Version = stringPtr("v1.0.0")
		current.ConnectionStatus.AppliedVersion = applied
		patched := false
		client := configureClientMock{getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil }, connectorVersions: connectorVersions{listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*testConnectorVersionList, error) {
			return &testConnectorVersionList{Items: []connectivity.ConnectorVersionSummary{{Version: "v1.1.0"}}}, nil
		}, getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
			return versionWithFileSchema(), nil
		}}, patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
			patched = true
			return &current, nil
		}}
		_, err := executeCommand(testNewConfigureCommand(factoryReturning(client), nil, nil), "stripe-eu", "--channel=stable", "--set=/etc/a=kept", "--confirm")
		if applied != nil {
			require.ErrorContains(t, err, "no candidate version satisfies channel")
			require.False(t, patched)
		} else {
			require.NoError(t, err)
			require.True(t, patched)
		}
	}
}

func TestConfigureOmitsUnchangedScalarsAndStoresCompleteReturnedInstance(t *testing.T) {
	current := instanceFixture("stripe-eu")
	setInstanceConfig(&current, &connectivity.ConnectionConfiguration{})
	returned := instanceFixture("stripe-eu")
	var gotPatch map[string]any
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
		patch: func(_ context.Context, _ string, patch map[string]any) (*connectivity.Connection, error) {
			gotPatch = patch
			return &returned, nil
		},
	}
	command := testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "stripe-eu", "--poll-interval=30s", "--confirm", "--output=json")

	require.NoError(t, err)
	require.Equal(t, map[string]any{"configuration": map[string]any{"pollInterval": "30s"}}, gotPatch)
	var envelope struct {
		Data ConfigureStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, returned, envelope.Data.Connection)
}

func TestConfigureRejectsNoChangesWithoutPatch(t *testing.T) {
	current := instanceFixture("stripe-eu")
	patched := false
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
		patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
			patched = true
			return nil, errors.New("PatchConnection must not be called")
		},
	}

	_, err := executeCommand(testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--confirm")

	require.EqualError(t, err, "no configuration changes requested")
	require.False(t, patched)
}

func TestConfigureConfirmationRejectionPreventsPatch(t *testing.T) {
	current := instanceFixture("stripe-eu")
	patched := false
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
		patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
			patched = true
			return nil, errors.New("PatchConnection must not be called")
		},
	}
	controller := NewConfigureController(mockReadFile(nil))
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }
	command := testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	require.NoError(t, command.ParseFlags([]string{"--poll-interval=30s"}))

	command.SetContext(context.Background())
	_, err := controller.run(command, []string{"stripe-eu"}, mockSDK(client))

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.False(t, patched)
}

func TestConfigureValidationAndArgumentsPreventPatch(t *testing.T) {
	t.Run("invalid configuration", func(t *testing.T) {
		current := instanceFixture("stripe-eu")
		patched := false
		client := configureClientMock{
			getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
			connectorVersions: connectorVersions{
				getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
					return versionWithFileSchema(), nil
				},
			},
			patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
				patched = true
				return nil, errors.New("PatchConnection must not be called")
			},
		}

		_, err := executeCommand(testNewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--set=UNKNOWN=value", "--confirm")

		require.ErrorContains(t, err, "unknown configuration keys: UNKNOWN")
		require.False(t, patched)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		usedFactory := false
		factory := func(*cobra.Command) (*sdk.FormanceV1, error) {
			usedFactory = true
			return nil, errors.New("factory must not be used")
		}
		command := testNewConfigureCommand(factory, mockReadFile(nil), mockPathCompleter(nil))

		_, err := executeCommand(command, "stripe-eu", "extra", "--poll-interval=30s", "--confirm")

		require.Error(t, err)
		require.False(t, usedFactory)
	})
}

func TestConfigurePreservesGetAndPatchAPIErrorsAndRejectsEmptyResponses(t *testing.T) {
	instanceError := sdkerrors.NewSDKError("API error occurred", 404, `{"code":"CONNECTORINSTANCE_NOT_FOUND","message":"missing"}`, nil)
	versionError := sdkerrors.NewSDKError("API error occurred", 404, `{"code":"CONNECTORVERSION_NOT_FOUND","message":"missing"}`, nil)
	patchError := sdkerrors.NewSDKError("API error occurred", 409, `{"code":"CONFLICT","message":"stale"}`, nil)
	current := instanceFixture("stripe-eu")
	tests := map[string]struct {
		client configureClientMock
		args   []string
		want   error
		text   string
	}{
		"instance API": {
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivity.Connection, error) {
				return nil, instanceError
			}},
			want: instanceError,
		},
		"empty instance": {
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivity.Connection, error) { return nil, nil }},
			text:   "empty response",
		},
		"version API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
				connectorVersions: connectorVersions{
					getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
						return nil, versionError
					},
				},
			},
			args: []string{"--set=API_URL=https://example"},
			want: versionError,
		},
		"empty version": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
				connectorVersions: connectorVersions{
					getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) { return nil, nil },
				},
			},
			args: []string{"--set=API_URL=https://example"},
			text: "empty response",
		},
		"patch API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
				patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
					return nil, patchError
				},
			},
			want: patchError,
		},
		"empty patch": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivity.Connection, error) { return &current, nil },
				patch: func(context.Context, string, map[string]any) (*connectivity.Connection, error) {
					return nil, nil
				},
			},
			text: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"stripe-eu", "--poll-interval=30s", "--confirm"}, test.args...)
			_, err := executeCommand(testNewConfigureCommand(factoryReturning(test.client), mockReadFile(nil), mockPathCompleter(nil)), args...)
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

func TestConfigureRegistersAliasesRootAndCompletions(t *testing.T) {
	current := instanceFixture("stripe-eu")
	var pathPrefix string
	client := configureClientMock{
		listInstances: func(_ context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
			require.Equal(t, connectivityinternal.ListOptions{PageSize: 100}, options)
			return &testConnectionList{Items: []connectivity.Connection{current}}, nil
		},
		getInstance: func(ctx context.Context, name string) (*connectivity.Connection, error) {
			require.True(t, hasTwoSecondDeadline(ctx))
			require.Equal(t, "stripe-eu", name)
			return &current, nil
		},
		connectorVersions: connectorVersions{
			listVersions: func(ctx context.Context, name string, _ connectivityinternal.ListOptions) (*testConnectorVersionList, error) {
				require.True(t, hasTwoSecondDeadline(ctx))
				require.Equal(t, "stripe", name)
				return versionListFixture(), nil
			},
			getVersion: func(ctx context.Context, name, _ string) (*connectivity.ConnectorVersion, error) {
				require.True(t, hasTwoSecondDeadline(ctx))
				require.Equal(t, "stripe", name)
				return versionWithFileSchema(), nil
			},
		},
	}
	factory := func(cmd *cobra.Command) (*sdk.FormanceV1, error) {
		require.True(t, hasTwoSecondDeadline(cmd.Context()))
		return mockSDK(client), nil
	}
	paths := func(prefix string) ([]string, error) {
		pathPrefix = prefix
		return []string{"fixtures/token.txt"}, nil
	}
	command := testNewConfigureCommand(factory, mockReadFile(nil), paths)

	require.Equal(t, "configure <connection>", command.Use)
	require.Equal(t, []string{"update", "c"}, command.Aliases)
	require.Error(t, command.Args(command, nil))
	require.NoError(t, command.Args(command, []string{"stripe-eu"}))
	require.Error(t, command.Args(command, []string{"stripe-eu", "extra"}))
	instances, directive := command.ValidArgsFunction(command, nil, "stripe")
	require.Equal(t, []string{"stripe-eu\tstripe · main"}, instances)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	versionCompletion, ok := command.GetFlagCompletionFunc(versionFlag)
	require.True(t, ok)
	require.NotNil(t, versionCompletion)
	versionCompletion = testcompleteVersions(factory, resolveConfigureConnector)
	versions, directive := versionCompletion(command, []string{"stripe-eu"}, "2")
	require.Equal(t, []string{"2.0.0"}, versions)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	setCompletion, ok := command.GetFlagCompletionFunc(setFlag)
	require.True(t, ok)
	require.NotNil(t, setCompletion)
	setCompletion = testCompleteSetValues(factory, resolveConfigureConnectorVersion, paths)
	sets, directive := setCompletion(command, []string{"stripe-eu"}, "/etc")
	require.Equal(t, []string{"/etc/a=\tPrimary config", "/etc/b=\tfile configuration"}, sets)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)
	sets, directive = setCompletion(command, []string{"stripe-eu"}, "/etc/a=@fixtures/t")
	require.Equal(t, "fixtures/t", pathPrefix)
	require.Equal(t, []string{"/etc/a=@fixtures/token.txt"}, sets)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	_, configRegistered := command.GetFlagCompletionFunc(configFlag)
	_, envFileRegistered := command.GetFlagCompletionFunc(envFileFlag)
	require.False(t, configRegistered)
	require.False(t, envFileRegistered)

	root := testNewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	child, _, err := root.Find([]string{"configure"})
	require.NoError(t, err)
	require.Equal(t, "configure", child.Name())
	require.True(t, reflect.DeepEqual([]string{"update", "c"}, child.Aliases))
}

func TestConfigureDoesNotExposeUnsupportedStartSequence(t *testing.T) {
	command := testNewConfigureCommand(nil, mockReadFile(nil), mockPathCompleter(nil))

	require.Nil(t, command.Flags().Lookup("start-sequence"))
}

func TestConfigureRejectsBindingFlagsBeforeClientUse(t *testing.T) {
	for _, flag := range []string{"connector", "ledger"} {
		t.Run(flag, func(t *testing.T) {
			usedFactory := false
			factory := func(*cobra.Command) (*sdk.FormanceV1, error) {
				usedFactory = true
				return nil, errors.New("client must not be used")
			}
			command := testNewConfigureCommand(factory, mockReadFile(nil), mockPathCompleter(nil))
			require.Nil(t, command.Flags().Lookup(flag))
			_, err := executeCommand(command, "stripe-eu", "--"+flag+"=main", "--confirm")
			require.ErrorContains(t, err, "unknown flag: --"+flag)
			require.False(t, usedFactory)
		})
	}
}
