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
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type configureClientMock struct {
	connectorVersions
	listInstances func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectionList, error)
	getInstance   func(context.Context, string) (*connectivityclient.Connection, error)
	patch         func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error)
}

func (m configureClientMock) ListConnections(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectionList, error) {
	return m.listInstances(ctx, options)
}

func (m configureClientMock) GetConnection(ctx context.Context, name string) (*connectivityclient.Connection, error) {
	return m.getInstance(ctx, name)
}

func (m configureClientMock) PatchConnection(ctx context.Context, name string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
	return m.patch(ctx, name, patch)
}

func TestConfigureBuildsNestedPatchFromCurrentConfigAndChangedScalars(t *testing.T) {
	current := instanceFixture("stripe-eu")
	setInstanceConfig(&current, &connectivityclient.ConnectionConfig{
		Env:   map[string]string{"API_URL": "https://current.example"},
		Files: map[string]string{"/etc/a": "old", "/etc/b": "kept"},
	})
	returned := instanceFixture("stripe-eu")
	var order []string
	var gotPatch connectivityclient.ConnectionPatch
	client := configureClientMock{
		getInstance: func(_ context.Context, name string) (*connectivityclient.Connection, error) {
			order = append(order, "instance:"+name)
			return &current, nil
		},
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
				order = append(order, "version:"+connector+"@"+version)
				return versionWithFileSchema(), nil
			},
		},
		patch: func(_ context.Context, name string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			order = append(order, "patch:"+name)
			gotPatch = patch
			return &returned, nil
		},
	}

	output, err := executeCommand(
		NewConfigureCommand(factoryReturning(client), mockReadFile(map[string]string{"new": "changed"}), mockPathCompleter(nil)),
		"stripe-eu", "--set=/etc/a=@new", "--version=3.0.0", "--ledger=main",
		"--poll-interval=15s", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, []string{"instance:stripe-eu", "version:stripe@3.0.0", "patch:stripe-eu"}, order,
		"the requested version supplies the schema the new configuration is validated against")
	spec, ok := gotPatch["configuration"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "3.0.0", spec["version"])
	require.Equal(t, "main", spec["ledger"])
	require.Equal(t, "15s", spec["pollInterval"])
	require.Len(t, spec, 5, "the immutable connector and unchanged fields must not be patched")
	require.NotContains(t, spec, "config")
	require.NotContains(t, spec, "env", "untouched environment entries must not be patched")
	require.Contains(t, spec, "channel")
	require.Nil(t, spec["channel"])
	require.Equal(t, map[string]any{"/etc/a": "changed"}, spec["files"], "untouched files must be omitted from merge patch")

	require.Equal(t, `Connection "stripe-eu" configured.`, strings.TrimSpace(output))
}

func TestConfigureWithoutVersionFlagUsesTheInstancePin(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Configuration.Version = stringPtr("1.4.2")
	setInstanceConfig(&current, &connectivityclient.ConnectionConfig{})
	var gotVersion string
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
				gotVersion = version
				return versionWithFileSchema(), nil
			},
		},
		patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			return &current, nil
		},
	}

	_, err := executeCommand(
		NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe-eu", "--set=/etc/a=kept", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "1.4.2", gotVersion)
}

func TestConfigureRequiresStoredOrExplicitSelector(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Configuration.Version = nil
	current.Configuration.Channel = nil
	current.Status.AppliedVersion = stringPtr("1.9.9")
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
		patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			t.Fatal("missing selector must not patch")
			return nil, nil
		},
	}
	_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--set=/etc/a=kept", "--confirm")
	require.ErrorContains(t, err, "no version/channel selector")
}

func TestConfigureRequiresUnambiguousStoredSelector(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Configuration.Channel = stringPtr("stable")
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
	}
	_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--set=/etc/a=kept", "--confirm")
	require.ErrorContains(t, err, "conflicting version/channel selectors")
}

func TestConfigureChannelTracksTheResolvedSameMajorHeadAcrossPages(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Configuration.Version = stringPtr("v1.0.0")
	current.Configuration.Channel = stringPtr("stable")
	current.Status.AppliedVersion = stringPtr("v1.0.0")
	setInstanceConfig(&current, &connectivityclient.ConnectionConfig{})
	var gotVersion string
	var gotPatch connectivityclient.ConnectionPatch
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
		connectorVersions: connectorVersions{
			listVersions: func(_ context.Context, _ string, options connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error) {
				switch options.Cursor {
				case "":
					return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{
						{Version: "v1.1.0"},
						{Version: "v1.2.0-beta.1"},
					}, HasMore: true, Next: "second"}, nil
				case "second":
					return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{
						{Version: "v1.3.0"},
						{Version: "v2.0.0"},
					}}, nil
				default:
					t.Fatalf("unexpected version cursor %q", options.Cursor)
					return nil, nil
				}
			},
			getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
				gotVersion = version
				return versionWithFileSchema(), nil
			},
		},
		patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			gotPatch = patch
			return &current, nil
		},
	}

	_, err := executeCommand(
		NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
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
		current.Configuration.Version = stringPtr("v1.0.0")
		current.Status.AppliedVersion = applied
		patched := false
		client := configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil }, connectorVersions: connectorVersions{listVersions: func(context.Context, string, connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error) {
			return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{{Version: "v1.1.0"}}}, nil
		}, getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
			return versionWithFileSchema(), nil
		}}, patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			patched = true
			return &current, nil
		}}
		_, err := executeCommand(NewConfigureCommand(factoryReturning(client), nil, nil), "stripe-eu", "--channel=stable", "--set=/etc/a=kept", "--confirm")
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
	setInstanceConfig(&current, &connectivityclient.ConnectionConfig{})
	returned := instanceFixture("stripe-eu")
	var gotPatch connectivityclient.ConnectionPatch
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
		patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			gotPatch = patch
			return &returned, nil
		},
	}
	command := NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "stripe-eu", "--ledger=main", "--confirm", "--output=json")

	require.NoError(t, err)
	require.Equal(t, connectivityclient.ConnectionPatch{"configuration": map[string]any{"ledger": "main"}}, gotPatch)
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
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
		patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			patched = true
			return nil, errors.New("PatchConnection must not be called")
		},
	}

	_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--confirm")

	require.EqualError(t, err, "no configuration changes requested")
	require.False(t, patched)
}

func TestConfigureConfirmationRejectionPreventsPatch(t *testing.T) {
	current := instanceFixture("stripe-eu")
	patched := false
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
		patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			patched = true
			return nil, errors.New("PatchConnection must not be called")
		},
	}
	controller := NewConfigureController(factoryReturning(client), mockReadFile(nil))
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }
	command := NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	require.NoError(t, command.ParseFlags([]string{"--ledger=main"}))

	_, err := controller.Run(command, []string{"stripe-eu"})

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.False(t, patched)
}

func TestConfigureValidationAndArgumentsPreventPatch(t *testing.T) {
	t.Run("invalid configuration", func(t *testing.T) {
		current := instanceFixture("stripe-eu")
		patched := false
		client := configureClientMock{
			getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
			connectorVersions: connectorVersions{
				getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
					return versionWithFileSchema(), nil
				},
			},
			patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
				patched = true
				return nil, errors.New("PatchConnection must not be called")
			},
		}

		_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--set=UNKNOWN=value", "--confirm")

		require.ErrorContains(t, err, "unknown configuration keys: UNKNOWN")
		require.False(t, patched)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		usedFactory := false
		factory := func(*cobra.Command) (connectivityclient.Client, error) {
			usedFactory = true
			return nil, errors.New("factory must not be used")
		}
		command := NewConfigureCommand(factory, mockReadFile(nil), mockPathCompleter(nil))

		_, err := executeCommand(command, "stripe-eu", "extra", "--ledger=main", "--confirm")

		require.Error(t, err)
		require.False(t, usedFactory)
	})
}

func TestConfigurePreservesGetAndPatchAPIErrorsAndRejectsEmptyResponses(t *testing.T) {
	instanceError := &connectivityclient.APIError{StatusCode: 404, Code: "CONNECTORINSTANCE_NOT_FOUND", Message: "missing"}
	versionError := &connectivityclient.APIError{StatusCode: 404, Code: "CONNECTORVERSION_NOT_FOUND", Message: "missing"}
	patchError := &connectivityclient.APIError{StatusCode: 409, Code: "CONFLICT", Message: "stale"}
	current := instanceFixture("stripe-eu")
	tests := map[string]struct {
		client configureClientMock
		args   []string
		want   error
		text   string
	}{
		"instance API": {
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.Connection, error) {
				return nil, instanceError
			}},
			want: instanceError,
		},
		"empty instance": {
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return nil, nil }},
			text:   "empty response",
		},
		"version API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				connectorVersions: connectorVersions{
					getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
						return nil, versionError
					},
				},
			},
			args: []string{"--set=API_URL=https://example"},
			want: versionError,
		},
		"empty version": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				connectorVersions: connectorVersions{
					getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) { return nil, nil },
				},
			},
			args: []string{"--set=API_URL=https://example"},
			text: "empty response",
		},
		"patch API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
					return nil, patchError
				},
			},
			want: patchError,
		},
		"empty patch": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
					return nil, nil
				},
			},
			text: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"stripe-eu", "--ledger=main", "--confirm"}, test.args...)
			_, err := executeCommand(NewConfigureCommand(factoryReturning(test.client), mockReadFile(nil), mockPathCompleter(nil)), args...)
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
		listInstances: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectionList, error) {
			require.Equal(t, connectivityclient.ListOptions{PageSize: 100}, options)
			return &connectivityclient.ConnectionList{Items: []connectivityclient.Connection{current}}, nil
		},
		getInstance: func(ctx context.Context, name string) (*connectivityclient.Connection, error) {
			require.True(t, connectivityinternal.IsNonInteractive(ctx))
			require.Equal(t, "stripe-eu", name)
			return &current, nil
		},
		connectorVersions: connectorVersions{
			listVersions: func(ctx context.Context, name string, _ connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error) {
				require.True(t, connectivityinternal.IsNonInteractive(ctx))
				require.Equal(t, "stripe", name)
				return versionListFixture(), nil
			},
			getVersion: func(ctx context.Context, name, _ string) (*connectivityclient.ConnectorVersion, error) {
				require.True(t, connectivityinternal.IsNonInteractive(ctx))
				require.Equal(t, "stripe", name)
				return versionWithFileSchema(), nil
			},
		},
	}
	factory := func(cmd *cobra.Command) (connectivityclient.Client, error) {
		require.True(t, connectivityinternal.IsNonInteractive(cmd.Context()))
		return client, nil
	}
	paths := func(prefix string) ([]string, error) {
		pathPrefix = prefix
		return []string{"fixtures/token.txt"}, nil
	}
	command := NewConfigureCommand(factory, mockReadFile(nil), paths)

	require.Equal(t, "configure <connection>", command.Use)
	require.Equal(t, []string{"config", "update", "c"}, command.Aliases)
	require.Error(t, command.Args(command, nil))
	require.NoError(t, command.Args(command, []string{"stripe-eu"}))
	require.Error(t, command.Args(command, []string{"stripe-eu", "extra"}))
	instances, directive := command.ValidArgsFunction(command, nil, "stripe")
	require.Equal(t, []string{"stripe-eu\tstripe · main"}, instances)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	versionCompletion, ok := command.GetFlagCompletionFunc(versionFlag)
	require.True(t, ok)
	versions, directive := versionCompletion(command, []string{"stripe-eu"}, "2")
	require.Equal(t, []string{"2.0.0"}, versions)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	setCompletion, ok := command.GetFlagCompletionFunc(setFlag)
	require.True(t, ok)
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

	root := NewCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	child, _, err := root.Find([]string{"configure"})
	require.NoError(t, err)
	require.Equal(t, "configure", child.Name())
	require.True(t, reflect.DeepEqual([]string{"config", "update", "c"}, child.Aliases))
}

func TestConfigureDoesNotExposeUnsupportedStartSequence(t *testing.T) {
	command := NewConfigureCommand(nil, mockReadFile(nil), mockPathCompleter(nil))

	require.Nil(t, command.Flags().Lookup("start-sequence"))
}

func TestConfigureAssignsMissingLedgerAndRejectsEmpty(t *testing.T) {
	for _, value := range []string{"main", ""} {
		t.Run(value, func(t *testing.T) {
			current := instanceFixture("imported")
			current.Configuration.Ledger = nil
			called := false
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
					called = true
					require.Equal(t, connectivityclient.ConnectionPatch{"configuration": map[string]any{"ledger": "main"}}, patch)
					return &current, nil
				},
			}
			_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "imported", "--ledger="+value, "--confirm")
			if value == "" {
				require.ErrorContains(t, err, "ledger must not be empty")
				require.False(t, called)
			} else {
				require.NoError(t, err)
				require.True(t, called)
			}
		})
	}
}

func TestConfigureConnectorFirstAssignmentSelectsRequestedSchema(t *testing.T) {
	for _, selector := range []string{"--version=v2.0.0", "--channel=stable"} {
		t.Run(selector, func(t *testing.T) {
			current := instanceFixture("imported")
			current.Configuration.Connector = ""
			current.Configuration.Ledger = nil
			current.Status = nil
			var selected string
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				connectorVersions: connectorVersions{
					listVersions: func(_ context.Context, name string, _ connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error) {
						require.Equal(t, "stripe", name)
						return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{{Version: "v2.0.0"}}}, nil
					},
					getVersion: func(_ context.Context, name, version string) (*connectivityclient.ConnectorVersion, error) {
						require.Equal(t, "stripe", name)
						selected = version
						return versionWithFileSchema(), nil
					},
				},
				patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
					spec := patch["configuration"].(map[string]any)
					require.Equal(t, "stripe", spec["connector"])
					require.Equal(t, "main", spec["ledger"])
					require.Equal(t, map[string]any{"/etc/a": "new"}, spec["files"])
					return &current, nil
				},
			}
			_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "imported", "--connector=stripe", "--ledger=main", selector, "--set=/etc/a=new", "--confirm")
			require.NoError(t, err)
			require.Equal(t, "v2.0.0", selected)
			command := NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
			require.NoError(t, command.ParseFlags([]string{"--connector=stripe", selector}))
			connector, err := resolveConfigureConnector(t.Context(), client, command, []string{"imported"})
			require.NoError(t, err)
			require.Equal(t, "stripe", connector)
			version, err := resolveConfigureConnectorVersion(t.Context(), client, command, []string{"imported"})
			require.NoError(t, err)
			require.NotNil(t, version)
			_, ok := command.GetFlagCompletionFunc(connectorFlag)
			require.True(t, ok)
		})
	}
}

func TestConfigureConnectorBoundIdentityGuard(t *testing.T) {
	for _, requested := range []string{"stripe", "different", ""} {
		t.Run(requested, func(t *testing.T) {
			current := instanceFixture("bound")
			called := false
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
					called = true
					require.Equal(t, connectivityclient.ConnectionPatch{"configuration": map[string]any{"connector": "stripe"}}, patch)
					return &current, nil
				},
			}
			_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "bound", "--connector="+requested, "--confirm")
			if requested == "stripe" {
				require.NoError(t, err)
				require.True(t, called)
			} else {
				require.Error(t, err)
				require.False(t, called)
			}
		})
	}
}
