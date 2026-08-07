package instances

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

type completionClientMock struct {
	connectivityclient.Client
	listInstances func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error)
	getPlugin     func(context.Context, string) (*connectivityclient.Plugin, error)
}

func (m completionClientMock) ListInstances(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
	return m.listInstances(ctx, options)
}

func (m completionClientMock) GetPlugin(ctx context.Context, name string) (*connectivityclient.Plugin, error) {
	return m.getPlugin(ctx, name)
}

func TestCompleteInstanceNamesUsesBoundedNonInteractiveQueryAndReturnsSortedPrefixMatchesWithDescriptions(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	var remaining time.Duration
	client := completionClientMock{listInstances: func(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
		gotOptions = options
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("completion context has no deadline")
		}
		remaining = time.Until(deadline)
		alpha := instanceFixture("alpha")
		alpha.Spec.Plugin = "stripe"
		alpha.Spec.Ledger = "eu"
		alpine := instanceFixture("alpine")
		alpine.Spec.Plugin = "wise"
		alpine.Spec.Ledger = "uk"
		return &connectivityclient.InstanceList{Items: []connectivityclient.Instance{
			instanceFixture("beta"), alpine, alpha,
		}}, nil
	}}
	completion := CompleteInstanceNames(func(cmd *cobra.Command) (connectivityclient.Client, error) {
		if !connectivityinternal.IsNonInteractive(cmd.Context()) {
			t.Fatal("completion factory context is interactive")
		}
		return client, nil
	})

	candidates, directive := completion(&cobra.Command{}, nil, "al")

	if !reflect.DeepEqual(gotOptions, connectivityclient.ListOptions{Limit: 500}) {
		t.Fatalf("ListInstances options = %#v, want limit 500", gotOptions)
	}
	if remaining <= 1500*time.Millisecond || remaining > 2*time.Second {
		t.Fatalf("completion deadline remaining = %s, want approximately 2s", remaining)
	}
	want := []string{"alpha\tstripe · eu", "alpine\twise · uk"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteVersionsUsesSelectedPluginAndReturnsSortedPrefixMatchesWithDescriptions(t *testing.T) {
	var gotName string
	client := completionClientMock{getPlugin: func(ctx context.Context, name string) (*connectivityclient.Plugin, error) {
		if !hasTwoSecondDeadline(ctx) {
			t.Fatal("version completion context does not have the expected deadline")
		}
		gotName = name
		plugin := pluginWithFileSchema()
		plugin.Spec.Versions = []connectivityclient.VersionEntry{
			{Version: "2.1.0", Digest: stringPtr("sha256:two")},
			{Version: "1.0.0", Image: stringPtr("registry/plugin:1")},
			{Version: "2.0.0", Image: stringPtr("registry/plugin:2")},
		}
		return plugin, nil
	}}
	completion := CompleteVersions(func(cmd *cobra.Command) (connectivityclient.Client, error) {
		if !connectivityinternal.IsNonInteractive(cmd.Context()) {
			t.Fatal("version completion factory context is interactive")
		}
		return client, nil
	}, func(_ *cobra.Command, args []string) string {
		return args[0]
	})

	candidates, directive := completion(&cobra.Command{}, []string{"stripe"}, "2")

	if gotName != "stripe" {
		t.Fatalf("GetPlugin name = %q, want stripe", gotName)
	}
	want := []string{"2.0.0\tregistry/plugin:2", "2.1.0\tsha256:two"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteSetValuesPrioritizesRequiredKeysDescribesAndOmitsSuppliedKeys(t *testing.T) {
	var resolvedClient connectivityclient.Client
	client := &completionClientMock{}
	resolvePlugin := func(ctx context.Context, got connectivityclient.Client, _ *cobra.Command, _ []string) (*connectivityclient.Plugin, error) {
		if !connectivityinternal.IsNonInteractive(ctx) || !hasTwoSecondDeadline(ctx) {
			t.Fatal("set completion resolver context must be non-interactive with a two-second deadline")
		}
		resolvedClient = got
		return pluginWithSchemaAndDefaults(), nil
	}
	completion := CompleteSetValues(func(cmd *cobra.Command) (connectivityclient.Client, error) {
		if !connectivityinternal.IsNonInteractive(cmd.Context()) {
			t.Fatal("set completion factory context is interactive")
		}
		return client, nil
	}, resolvePlugin, mockPathCompleter(nil))
	command := &cobra.Command{}
	command.Flags().StringArray("set", nil, "")
	if err := command.Flags().Set("set", "TOKEN=secret://plugin/token"); err != nil {
		t.Fatal(err)
	}

	candidates, directive := completion(command, nil, "")

	if resolvedClient != client {
		t.Fatal("resolver did not receive the factory client")
	}
	want := []string{
		"/etc/plugin/config.yaml=\tPlugin configuration",
		"API_URL=\tAPI endpoint",
		"/etc/plugin/ca.pem=\tfile configuration",
		"TIMEOUT=\tenvironment configuration",
	}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want required then optional keys %#v", candidates, want)
	}
	wantDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	if directive != wantDirective {
		t.Fatalf("directive = %v, want NoFileComp|NoSpace", directive)
	}
}

func TestCompleteSetValuesPreservesKeyAtPrefixForInjectedPathCandidates(t *testing.T) {
	var gotPrefix string
	paths := func(prefix string) ([]string, error) {
		gotPrefix = prefix
		return []string{"fixtures/alpha/", "fixtures/api-key.txt"}, nil
	}
	completion := CompleteSetValues(nil, nil, paths)

	candidates, directive := completion(&cobra.Command{}, nil, "API_KEY=@fixtures/a")

	if gotPrefix != "fixtures/a" {
		t.Fatalf("path prefix = %q, want fixtures/a", gotPrefix)
	}
	want := []string{"API_KEY=@fixtures/alpha/", "API_KEY=@fixtures/api-key.txt"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want prefixed paths %#v", candidates, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteFunctionsReturnSilentlyOnFactoryAPIResolverPathAndTimeoutErrors(t *testing.T) {
	apiErrorClient := completionClientMock{
		listInstances: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
			return nil, errors.New("unsupported deployment")
		},
		getPlugin: func(context.Context, string) (*connectivityclient.Plugin, error) {
			return nil, errors.New("unsupported deployment")
		},
	}
	tests := map[string]struct {
		completion cobra.CompletionFunc
		command    *cobra.Command
		args       []string
		prefix     string
	}{
		"instance authentication": {
			completion: CompleteInstanceNames(func(*cobra.Command) (connectivityclient.Client, error) { return nil, errors.New("not authenticated") }),
			command:    &cobra.Command{},
		},
		"instance nil client": {
			completion: CompleteInstanceNames(func(*cobra.Command) (connectivityclient.Client, error) { return nil, nil }),
			command:    &cobra.Command{},
		},
		"instance API": {completion: CompleteInstanceNames(factoryReturning(apiErrorClient)), command: &cobra.Command{}},
		"instance timeout": {
			completion: CompleteInstanceNames(factoryReturning(completionClientMock{listInstances: func(ctx context.Context, _ connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}})),
			command: commandWithExpiredContext(),
		},
		"version empty plugin": {completion: CompleteVersions(factoryReturning(apiErrorClient), func(*cobra.Command, []string) string { return "" }), command: &cobra.Command{}},
		"version API":          {completion: CompleteVersions(factoryReturning(apiErrorClient), func(*cobra.Command, []string) string { return "stripe" }), command: &cobra.Command{}},
		"set resolver": {
			completion: CompleteSetValues(factoryReturning(completionClientMock{}), func(context.Context, connectivityclient.Client, *cobra.Command, []string) (*connectivityclient.Plugin, error) {
				return nil, errors.New("cannot resolve plugin")
			}, mockPathCompleter(nil)),
			command: &cobra.Command{},
		},
		"path": {
			completion: CompleteSetValues(nil, nil, func(string) ([]string, error) { return nil, errors.New("cannot read directory") }),
			command:    &cobra.Command{},
			prefix:     "API_KEY=@fixtures/a",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			candidates, directive := test.completion(test.command, test.args, test.prefix)
			if len(candidates) != 0 {
				t.Fatalf("candidates = %#v, want none", candidates)
			}
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("directive = %v, want NoFileComp", directive)
			}
		})
	}
}

func TestCompletionClientCopyDoesNotMutateOriginalCommandContext(t *testing.T) {
	originalContext := context.WithValue(context.Background(), struct{}{}, "original")
	command := &cobra.Command{}
	command.SetContext(originalContext)
	completion := CompleteInstanceNames(func(cmd *cobra.Command) (connectivityclient.Client, error) {
		if !connectivityinternal.IsNonInteractive(cmd.Context()) {
			t.Fatal("copied completion command must be non-interactive")
		}
		return completionClientMock{listInstances: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.InstanceList, error) {
			return &connectivityclient.InstanceList{}, nil
		}}, nil
	})

	_, _ = completion(command, nil, "")

	if command.Context() != originalContext {
		t.Fatal("completion mutated the original command context")
	}
	if connectivityinternal.IsNonInteractive(command.Context()) {
		t.Fatal("original command context became non-interactive")
	}
}

func hasTwoSecondDeadline(ctx context.Context) bool {
	deadline, ok := ctx.Deadline()
	if !ok {
		return false
	}
	remaining := time.Until(deadline)
	return remaining > 1500*time.Millisecond && remaining <= 2*time.Second
}

func commandWithExpiredContext() *cobra.Command {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := &cobra.Command{}
	command.SetContext(ctx)
	return command
}
