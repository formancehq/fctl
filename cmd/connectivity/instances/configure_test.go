package instances

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
	connectivityclient.Client
	listInstances func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error)
	getInstance   func(context.Context, string) (*connectivityclient.Instance, error)
	getPlugin     func(context.Context, string) (*connectivityclient.Plugin, error)
	patch         func(context.Context, string, connectivityclient.InstancePatch) (*connectivityclient.Instance, error)
}

func (m configureClientMock) ListInstances(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
	return m.listInstances(ctx, options)
}

func (m configureClientMock) GetInstance(ctx context.Context, name string) (*connectivityclient.Instance, error) {
	return m.getInstance(ctx, name)
}

func (m configureClientMock) GetPlugin(ctx context.Context, name string) (*connectivityclient.Plugin, error) {
	return m.getPlugin(ctx, name)
}

func (m configureClientMock) PatchInstance(ctx context.Context, name string, patch connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
	return m.patch(ctx, name, patch)
}

func TestConfigureBuildsNestedPatchFromCurrentConfigAndChangedScalars(t *testing.T) {
	mode := int32(384)
	current := instanceFixture("stripe-eu")
	current.Spec.Config = &connectivityclient.InstanceConfig{
		Env: map[string]connectivityclient.EnvValue{"API_URL": {Value: stringPtr("https://current.example")}},
		Files: []connectivityclient.FileMount{
			{Path: "/etc/a", Value: stringPtr("old"), Mode: &mode},
			{Path: "/etc/b", SecretRef: &connectivityclient.KeyRef{Name: "plugin-secrets", Key: "b"}},
		},
	}
	plugin := pluginWithFileSchema()
	plugin.Metadata.Name = stringPtr("stripe")
	plugin.Spec.Defaults = &connectivityclient.InstanceConfig{Files: []connectivityclient.FileMount{{Path: "/etc/b", Value: stringPtr("must-not-replace-current")}}}
	returned := instanceFixture("stripe-eu")
	var order []string
	var gotPatch connectivityclient.InstancePatch
	client := configureClientMock{
		getInstance: func(_ context.Context, name string) (*connectivityclient.Instance, error) {
			order = append(order, "instance:"+name)
			return &current, nil
		},
		getPlugin: func(_ context.Context, name string) (*connectivityclient.Plugin, error) {
			order = append(order, "plugin:"+name)
			return plugin, nil
		},
		patch: func(_ context.Context, name string, patch connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
			order = append(order, "patch:"+name)
			gotPatch = patch
			return &returned, nil
		},
	}

	output, err := executeCommand(
		NewConfigureCommand(factoryReturning(client), mockReadFile(map[string]string{"new": "changed"}), mockPathCompleter(nil)),
		"stripe-eu", "--set=/etc/a=@new", "--version=3.0.0", "--ledger=archive",
		"--start-sequence=0", "--poll-interval=15s", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, []string{"instance:stripe-eu", "plugin:stripe", "patch:stripe-eu"}, order)
	spec, ok := gotPatch["spec"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "3.0.0", spec["version"])
	require.Equal(t, "archive", spec["ledger"])
	require.Equal(t, int64(0), spec["startSequence"])
	require.Equal(t, "15s", spec["pollInterval"])
	require.Len(t, spec, 5, "the immutable plugin and unchanged fields must not be patched")
	config, ok := spec["config"].(*connectivityclient.InstanceConfig)
	require.True(t, ok)
	require.Equal(t, "https://current.example", *config.Env["API_URL"].Value, "current config must win over plugin defaults")
	require.Len(t, config.Files, 2)
	require.Equal(t, "/etc/a", config.Files[0].Path)
	require.Equal(t, "changed", *config.Files[0].Value)
	require.Equal(t, mode, *config.Files[0].Mode)
	require.Equal(t, current.Spec.Config.Files[1], config.Files[1], "untouched file mounts must be preserved")
	require.Equal(t, `Instance "stripe-eu" configured.`, strings.TrimSpace(output))
}

func TestConfigureOmitsUnchangedScalarsAndStoresCompleteReturnedInstance(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Spec.Config = &connectivityclient.InstanceConfig{}
	returned := instanceFixture("stripe-eu")
	returned.Metadata.Labels = map[string]string{"region": "eu"}
	var gotPatch connectivityclient.InstancePatch
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
		getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
			return &connectivityclient.Plugin{}, nil
		},
		patch: func(_ context.Context, _ string, patch connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
			gotPatch = patch
			return &returned, nil
		},
	}
	command := NewConfigureCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "stripe-eu", "--ledger=archive", "--confirm", "--output=json")

	require.NoError(t, err)
	require.Equal(t, connectivityclient.InstancePatch{"spec": map[string]any{"ledger": "archive"}}, gotPatch)
	var envelope struct {
		Data ConfigureStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, returned, envelope.Data.Instance)
}

func TestConfigureRejectsNoChangesWithoutPatch(t *testing.T) {
	current := instanceFixture("stripe-eu")
	patched := false
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
		getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
			return &connectivityclient.Plugin{}, nil
		},
		patch: func(context.Context, string, connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
			patched = true
			return nil, errors.New("PatchInstance must not be called")
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
		getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
		getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
			return &connectivityclient.Plugin{}, nil
		},
		patch: func(context.Context, string, connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
			patched = true
			return nil, errors.New("PatchInstance must not be called")
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
			getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
			getPlugin:   func(context.Context, string) (*connectivityclient.Plugin, error) { return pluginWithFileSchema(), nil },
			patch: func(context.Context, string, connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
				patched = true
				return nil, errors.New("PatchInstance must not be called")
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
	instanceError := &connectivityclient.APIError{StatusCode: 404, Code: "INSTANCE_NOT_FOUND", Message: "missing"}
	pluginError := &connectivityclient.APIError{StatusCode: 404, Code: "PLUGIN_NOT_FOUND", Message: "missing"}
	patchError := &connectivityclient.APIError{StatusCode: 409, Code: "CONFLICT", Message: "stale"}
	current := instanceFixture("stripe-eu")
	tests := map[string]struct {
		client configureClientMock
		want   error
		text   string
	}{
		"instance API": {
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return nil, instanceError }},
			want:   instanceError,
		},
		"empty instance": {
			client: configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return nil, nil }},
			text:   "empty instance response",
		},
		"plugin API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
				getPlugin:   func(context.Context, string) (*connectivityclient.Plugin, error) { return nil, pluginError },
			},
			want: pluginError,
		},
		"empty plugin": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
				getPlugin:   func(context.Context, string) (*connectivityclient.Plugin, error) { return nil, nil },
			},
			text: "empty plugin response",
		},
		"patch API": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
				getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
					return &connectivityclient.Plugin{}, nil
				},
				patch: func(context.Context, string, connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
					return nil, patchError
				},
			},
			want: patchError,
		},
		"empty patch": {
			client: configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Instance, error) { return &current, nil },
				getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
					return &connectivityclient.Plugin{}, nil
				},
				patch: func(context.Context, string, connectivityclient.InstancePatch) (*connectivityclient.Instance, error) {
					return nil, nil
				},
			},
			text: "empty instance response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(NewConfigureCommand(factoryReturning(test.client), mockReadFile(nil), mockPathCompleter(nil)), "stripe-eu", "--ledger=archive", "--confirm")
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
	plugin := pluginWithFileSchema()
	plugin.Metadata.Name = stringPtr("stripe")
	current := instanceFixture("stripe-eu")
	var pathPrefix string
	client := configureClientMock{
		listInstances: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
			require.Equal(t, connectivityclient.ListOptions{Limit: 500}, options)
			return &connectivityclient.InstanceList{Items: []connectivityclient.Instance{current}}, nil
		},
		getInstance: func(ctx context.Context, name string) (*connectivityclient.Instance, error) {
			require.True(t, connectivityinternal.IsNonInteractive(ctx))
			require.Equal(t, "stripe-eu", name)
			return &current, nil
		},
		getPlugin: func(ctx context.Context, name string) (*connectivityclient.Plugin, error) {
			require.True(t, connectivityinternal.IsNonInteractive(ctx))
			require.Equal(t, "stripe", name)
			return plugin, nil
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

	require.Equal(t, "configure <instance>", command.Use)
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
	require.Equal(t, []string{"2.0.0\texample/plugin:2.0.0"}, versions)
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
