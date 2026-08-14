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

type configureClientMock struct {
	connectorVersions
	listInstances func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error)
	getInstance   func(context.Context, string) (*connectivityclient.ConnectorInstance, error)
	patch         func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error)
}

func (m configureClientMock) ListConnectorInstances(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
	return m.listInstances(ctx, options)
}

func (m configureClientMock) GetConnectorInstance(ctx context.Context, name string) (*connectivityclient.ConnectorInstance, error) {
	return m.getInstance(ctx, name)
}

func (m configureClientMock) PatchConnectorInstance(ctx context.Context, name string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
	return m.patch(ctx, name, patch)
}

func TestConfigureBuildsNestedPatchFromCurrentConfigAndChangedScalars(t *testing.T) {
	mode := int32(384)
	current := instanceFixture("stripe-eu")
	current.Spec.Config = &connectivityclient.ConnectorInstanceConfig{
		Env: map[string]connectivityclient.EnvValue{"API_URL": {Value: stringPtr("https://current.example")}},
		Files: []connectivityclient.FileMount{
			{Path: "/etc/a", Value: stringPtr("old"), Mode: &mode},
			{Path: "/etc/b", SecretRef: &connectivityclient.KeyRef{Name: "connector-secrets", Key: "b"}},
		},
	}
	returned := instanceFixture("stripe-eu")
	var order []string
	var gotPatch connectivityclient.ConnectorInstancePatch
	client := configureClientMock{
		getInstance: func(_ context.Context, name string) (*connectivityclient.ConnectorInstance, error) {
			order = append(order, "instance:"+name)
			return &current, nil
		},
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
				order = append(order, "version:"+connector+"@"+version)
				return versionWithFileSchema(), nil
			},
		},
		patch: func(_ context.Context, name string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
			order = append(order, "patch:"+name)
			gotPatch = patch
			return &returned, nil
		},
	}

	output, err := executeCommand(
		NewConfigureCommand(factoryReturning(client), mockReadFile(map[string]string{"new": "changed"}), mockPathCompleter(nil)),
		"stripe-eu", "--set=/etc/a=@new", "--version=3.0.0", "--ledger=archive",
		"--poll-interval=15s", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, []string{"instance:stripe-eu", "version:stripe@3.0.0", "patch:stripe-eu"}, order,
		"the requested version supplies the schema the new configuration is validated against")
	spec, ok := gotPatch["spec"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "3.0.0", spec["version"])
	require.Equal(t, "archive", spec["ledger"])
	require.Equal(t, "15s", spec["pollInterval"])
	require.Len(t, spec, 4, "the immutable connector and unchanged fields must not be patched")
	config, ok := spec["config"].(*connectivityclient.ConnectorInstanceConfig)
	require.True(t, ok)
	require.Equal(t, "https://current.example", *config.Env["API_URL"].Value, "the current configuration is the patch base")
	require.Len(t, config.Files, 2)
	require.Equal(t, "/etc/a", config.Files[0].Path)
	require.Equal(t, "changed", *config.Files[0].Value)
	require.Equal(t, mode, *config.Files[0].Mode)
	require.Equal(t, current.Spec.Config.Files[1], config.Files[1], "untouched file mounts must be preserved")
	require.Equal(t, `Connector instance "stripe-eu" configured.`, strings.TrimSpace(output))
}

func TestConfigureWithoutVersionFlagUsesTheInstancePin(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Spec.Version = stringPtr("1.4.2")
	current.Spec.Config = &connectivityclient.ConnectorInstanceConfig{}
	var gotVersion string
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
				gotVersion = version
				return versionWithFileSchema(), nil
			},
		},
		patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
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

func TestConfigureFallsBackToTheAppliedVersionWhenUnpinned(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Spec.Version = nil
	current.Status.ResolvedVersion = stringPtr("1.9.9")
	current.Spec.Config = &connectivityclient.ConnectorInstanceConfig{}
	var gotVersion string
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, _, version string) (*connectivityclient.ConnectorVersion, error) {
				gotVersion = version
				return versionWithFileSchema(), nil
			},
		},
		patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
			return &current, nil
		},
	}

	_, err := executeCommand(
		NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)),
		"stripe-eu", "--set=/etc/a=kept", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "1.9.9", gotVersion)
}

func TestConfigureChannelTracksTheResolvedSameMajorHeadAcrossPages(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Spec.Version = stringPtr("v1.0.0")
	current.Spec.Channel = stringPtr("stable")
	current.Status.ResolvedVersion = stringPtr("v1.0.0")
	current.Spec.Config = &connectivityclient.ConnectorInstanceConfig{}
	var gotVersion string
	var gotPatch connectivityclient.ConnectorInstancePatch
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		connectorVersions: connectorVersions{
			listVersions: func(_ context.Context, _ string, options connectivityclient.ListOptions) (*connectivityclient.ConnectorVersionList, error) {
				switch options.Cursor {
				case "":
					return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{
						{Version: "v1.1.0", Image: "example:v1.1.0"},
						{Version: "v1.2.0-beta.1", Image: "example:v1.2.0-beta.1"},
					}, HasMore: true, Next: "second"}, nil
				case "second":
					return &connectivityclient.ConnectorVersionList{Items: []connectivityclient.ConnectorVersionSummary{
						{Version: "v1.3.0", Image: "example:v1.3.0"},
						{Version: "v2.0.0", Image: "example:v2.0.0"},
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
		patch: func(_ context.Context, _ string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
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
	spec := gotPatch["spec"].(map[string]any)
	require.Equal(t, "stable", spec["channel"])
	require.Contains(t, spec, "version", "switching to a channel removes an existing pin")
	require.Nil(t, spec["version"])
}

func TestConfigureOmitsUnchangedScalarsAndStoresCompleteReturnedInstance(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Spec.Config = &connectivityclient.ConnectorInstanceConfig{}
	returned := instanceFixture("stripe-eu")
	returned.Metadata.Labels = map[string]string{"region": "eu"}
	var gotPatch connectivityclient.ConnectorInstancePatch
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		patch: func(_ context.Context, _ string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
			gotPatch = patch
			return &returned, nil
		},
	}
	command := NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "stripe-eu", "--ledger=archive", "--confirm", "--output=json")

	require.NoError(t, err)
	require.Equal(t, connectivityclient.ConnectorInstancePatch{"spec": map[string]any{"ledger": "archive"}}, gotPatch)
	var envelope struct {
		Data ConfigureStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, returned, envelope.Data.ConnectorInstance)
}

func TestConfigureRejectsNoChangesWithoutPatch(t *testing.T) {
	current := instanceFixture("stripe-eu")
	patched := false
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
			patched = true
			return nil, errors.New("PatchConnectorInstance must not be called")
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
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
			patched = true
			return nil, errors.New("PatchConnectorInstance must not be called")
		},
	}
	controller := NewConfigureController(factoryReturning(client), mockReadFile(nil))
	controller.approve = func(*cobra.Command, string, ...any) bool { return false }
	command := NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	require.NoError(t, command.ParseFlags([]string{"--ledger=archive"}))

	_, err := controller.Run(command, []string{"stripe-eu"})

	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.False(t, patched)
}

func TestConfigureValidationAndArgumentsPreventPatch(t *testing.T) {
	t.Run("invalid configuration", func(t *testing.T) {
		current := instanceFixture("stripe-eu")
		patched := false
		client := configureClientMock{
			getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
			connectorVersions: connectorVersions{
				getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
					return versionWithFileSchema(), nil
				},
			},
			patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
				patched = true
				return nil, errors.New("PatchConnectorInstance must not be called")
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

		_, err := executeCommand(command, "stripe-eu", "extra", "--ledger=archive", "--confirm")

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
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) {
				return nil, instanceError
			}},
			want: instanceError,
		},
		"empty instance": {
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return nil, nil }},
			text:   "empty response",
		},
		"version API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
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
				getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
				connectorVersions: connectorVersions{
					getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) { return nil, nil },
				},
			},
			args: []string{"--set=API_URL=https://example"},
			text: "empty response",
		},
		"patch API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
				patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
					return nil, patchError
				},
			},
			want: patchError,
		},
		"empty patch": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
				patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
					return nil, nil
				},
			},
			text: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"stripe-eu", "--ledger=archive", "--confirm"}, test.args...)
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
		listInstances: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorInstanceList, error) {
			require.Equal(t, connectivityclient.ListOptions{PageSize: 100}, options)
			return &connectivityclient.ConnectorInstanceList{Items: []connectivityclient.ConnectorInstance{current}}, nil
		},
		getInstance: func(ctx context.Context, name string) (*connectivityclient.ConnectorInstance, error) {
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

	require.Equal(t, "configure <connectorinstance>", command.Use)
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
	require.Equal(t, []string{"2.0.0\texample/connector:2.0.0"}, versions)
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
