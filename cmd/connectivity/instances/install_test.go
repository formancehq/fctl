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

type mutationClientMock struct {
	connectivityclient.Client
	listPlugins func(context.Context, connectivityclient.ListOptions) (*connectivityclient.PluginList, error)
	getPlugin   func(context.Context, string) (*connectivityclient.Plugin, error)
	create      func(context.Context, connectivityclient.InstanceCreate) (*connectivityclient.Instance, error)
}

func (m mutationClientMock) ListPlugins(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
	return m.listPlugins(ctx, options)
}

func (m mutationClientMock) GetPlugin(ctx context.Context, name string) (*connectivityclient.Plugin, error) {
	return m.getPlugin(ctx, name)
}

func (m mutationClientMock) CreateInstance(ctx context.Context, body connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
	return m.create(ctx, body)
}

func TestInstallBuildsInstanceFromPluginSchemaAndOnlySetsChangedScalars(t *testing.T) {
	plugin := pluginWithSchemaAndDefaults()
	var gotPlugin string
	var gotBody connectivityclient.InstanceCreate
	client := mutationClientMock{
		getPlugin: func(_ context.Context, name string) (*connectivityclient.Plugin, error) {
			gotPlugin = name
			return plugin, nil
		},
		create: func(_ context.Context, body connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
			gotBody = body
			return &connectivityclient.Instance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(body.Name)}, Spec: body.Spec}, nil
		},
	}
	read := mockReadFile(map[string]string{
		"install.yaml": "env:\n  API_URL:\n    value: https://config.example\n",
		"plugin.env":   "TIMEOUT=45s\n",
		"token.txt":    "file-token",
	})

	output, err := executeCommand(
		NewInstallCommand(factoryReturning(client), read, mockPathCompleter(nil)),
		"stripe", "--name=stripe-eu", "--ledger=main", "--version=2.0.0",
		"--start-sequence=0", "--poll-interval=3s", "--config=install.yaml",
		"--env-file=plugin.env", "--set=TOKEN=@token.txt", "--confirm",
	)

	require.NoError(t, err)
	require.Equal(t, "stripe", gotPlugin)
	require.Equal(t, "stripe-eu", gotBody.Name)
	require.Equal(t, "stripe", gotBody.Spec.Plugin)
	require.Equal(t, "main", gotBody.Spec.Ledger)
	require.Equal(t, "2.0.0", *gotBody.Spec.Version)
	require.Equal(t, int64(0), *gotBody.Spec.StartSequence)
	require.Equal(t, "3s", *gotBody.Spec.PollInterval)
	require.Equal(t, "https://config.example", *gotBody.Spec.Config.Env["API_URL"].Value)
	require.Equal(t, "file-token", *gotBody.Spec.Config.Env["TOKEN"].Value)
	require.Equal(t, "45s", *gotBody.Spec.Config.Env["TIMEOUT"].Value)
	require.Equal(t, plugin.Spec.Defaults.Files, gotBody.Spec.Config.Files)
	require.Equal(t, "Instance \"stripe-eu\" installed with plugin \"stripe\" for ledger \"main\".", strings.TrimSpace(output))
}

func TestInstallDefaultsNameToPluginAndOmitsUnsetOptionalData(t *testing.T) {
	plugin := &connectivityclient.Plugin{}
	var gotBody connectivityclient.InstanceCreate
	client := mutationClientMock{
		getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) { return plugin, nil },
		create: func(_ context.Context, body connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
			gotBody = body
			return &connectivityclient.Instance{Metadata: connectivityclient.ObjectMeta{Name: stringPtr(body.Name)}, Spec: body.Spec}, nil
		},
	}

	_, err := executeCommand(NewInstallCommand(factoryReturning(client), mockReadFile(nil), mockPathCompleter(nil)), "wise", "--ledger=main", "--confirm")

	require.NoError(t, err)
	require.Equal(t, "wise", gotBody.Name)
	require.Nil(t, gotBody.Spec.Version)
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

func TestInstallConfirmationRejectionPreventsCreateInstance(t *testing.T) {
	created := false
	client := mutationClientMock{
		getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
			return &connectivityclient.Plugin{}, nil
		},
		create: func(context.Context, connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
			created = true
			return nil, errors.New("CreateInstance must not be called")
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

func TestInstallPreservesPluginAndCreateAPIErrorsAndRejectsEmptyResponses(t *testing.T) {
	pluginError := &connectivityclient.APIError{StatusCode: 404, Code: "PLUGIN_NOT_FOUND", Message: "missing"}
	createError := &connectivityclient.APIError{StatusCode: 409, Code: "INSTANCE_EXISTS", Message: "duplicate"}
	tests := map[string]struct {
		client mutationClientMock
		want   error
		text   string
	}{
		"plugin API": {
			client: mutationClientMock{getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) { return nil, pluginError }},
			want:   pluginError,
		},
		"empty plugin": {
			client: mutationClientMock{getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) { return nil, nil }},
			text:   "empty plugin response",
		},
		"create API": {
			client: mutationClientMock{
				getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
					return &connectivityclient.Plugin{}, nil
				},
				create: func(context.Context, connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
					return nil, createError
				},
			},
			want: createError,
		},
		"empty create": {
			client: mutationClientMock{
				getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
					return &connectivityclient.Plugin{}, nil
				},
				create: func(context.Context, connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
					return nil, nil
				},
			},
			text: "empty instance response",
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

func TestInstallJSONStoresCompleteReturnedInstance(t *testing.T) {
	returned := instanceFixture("stripe-eu")
	returned.Metadata.Labels = map[string]string{"region": "eu"}
	returned.Spec.Config = &connectivityclient.InstanceConfig{Env: map[string]connectivityclient.EnvValue{
		"TOKEN": {Value: stringPtr("preserved-in-json")},
	}}
	client := mutationClientMock{
		getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
			return &connectivityclient.Plugin{}, nil
		},
		create: func(context.Context, connectivityclient.InstanceCreate) (*connectivityclient.Instance, error) {
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
	require.Equal(t, returned, envelope.Data.Instance)
}

func TestInstallRegistersAliasesAndRootIntegration(t *testing.T) {
	standalone := NewInstallCommand(nil, mockReadFile(nil), mockPathCompleter(nil))
	require.Equal(t, "install <plugin>", standalone.Use)
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

func TestInstallWiresPluginVersionSetAndFileCompletions(t *testing.T) {
	plugin := pluginWithSchemaAndDefaults()
	plugin.Metadata.Name = stringPtr("stripe")
	plugin.Spec.Description = stringPtr("Stripe ingestion")
	client := mutationClientMock{
		listPlugins: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
			require.Equal(t, connectivityclient.ListOptions{Limit: 500}, options)
			return &connectivityclient.PluginList{Items: []connectivityclient.Plugin{*plugin}}, nil
		},
		getPlugin: func(ctx context.Context, name string) (*connectivityclient.Plugin, error) {
			require.True(t, connectivityinternal.IsNonInteractive(ctx))
			require.Equal(t, "stripe", name)
			return plugin, nil
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

	plugins, directive := command.ValidArgsFunction(command, nil, "str")
	require.Equal(t, []string{"stripe\tStripe ingestion"}, plugins)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	versionCompletion, ok := command.GetFlagCompletionFunc("version")
	require.True(t, ok)
	versions, directive := versionCompletion(command, []string{"stripe"}, "2")
	require.Equal(t, []string{"2.0.0\texample/plugin:2.0.0"}, versions)
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
