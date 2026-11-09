package connections

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type completionClientMock struct {
	connectorVersions
	listInstances func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error)
}

func TestCompleteConnectionNamesUsesBoundedNonInteractiveQueryAndReturnsSortedPrefixMatchesWithDescriptions(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	var remaining time.Duration
	client := completionClientMock{listInstances: func(ctx context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
		gotOptions = options
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("completion context has no deadline")
		}
		remaining = time.Until(deadline)
		alpha := instanceFixture("alpha")
		alpha.ConnectionConfiguration.Connector = stringPtr("stripe")
		alpha.ConnectionConfiguration.Ledger = stringPtr("eu")
		alpine := instanceFixture("alpine")
		alpine.ConnectionConfiguration.Connector = stringPtr("wise")
		alpine.ConnectionConfiguration.Ledger = stringPtr("uk")
		return &testConnectionList{Items: []connectivity.Connection{
			instanceFixture("beta"), alpine, alpha,
		}}, nil
	}}
	completion := testCompleteConnectionNames(func(cmd *cobra.Command) (*sdk.FormanceV1, error) {
		if !hasTwoSecondDeadline(cmd.Context()) {
			t.Fatal("completion factory context is interactive")
		}
		return mockSDK(client), nil
	})

	candidates, directive := completion(&cobra.Command{}, nil, "al")

	if !reflect.DeepEqual(gotOptions, connectivityinternal.ListOptions{PageSize: 100}) {
		t.Fatalf("ListConnections options = %#v, want page size 100", gotOptions)
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

func TestCompleteVersionsUsesSelectedConnectorAndReturnsSortedPrefixMatchesWithDescriptions(t *testing.T) {
	var gotName string
	client := completionClientMock{connectorVersions: connectorVersions{
		listVersions: func(ctx context.Context, name string, options connectivityinternal.ListOptions) (*testConnectorVersionList, error) {
			if !hasTwoSecondDeadline(ctx) {
				t.Fatal("version completion context does not have the expected deadline")
			}
			if options.PageSize != 100 {
				t.Fatalf("ListConnectorVersions page size = %d, want 100", options.PageSize)
			}
			gotName = name
			return &testConnectorVersionList{Items: []connectivity.ConnectorVersionSummary{
				{Version: "2.1.0", Digest: stringPtr("sha256:two")},
				{Version: "1.0.0"},
				{Version: "2.0.0"},
			}}, nil
		},
	}}
	completion := testCompleteVersions(func(cmd *cobra.Command) (*sdk.FormanceV1, error) {
		if !hasTwoSecondDeadline(cmd.Context()) {
			t.Fatal("version completion factory context is interactive")
		}
		return mockSDK(client), nil
	}, func(_ *cobra.Command, args []string) string {
		return args[0]
	})

	candidates, directive := completion(&cobra.Command{}, []string{"stripe"}, "2")

	if gotName != "stripe" {
		t.Fatalf("ListConnectorVersions connector = %q, want stripe", gotName)
	}
	want := []string{"2.1.0\tlatest · sha256:two", "2.0.0"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteSetValuesPrioritizesRequiredKeysDescribesAndOmitsSuppliedKeys(t *testing.T) {
	var resolvedClient *sdk.FormanceV1
	client := mockSDK(&completionClientMock{})
	resolveVersion := func(ctx context.Context, got *sdk.FormanceV1, _ *cobra.Command, _ []string) (*connectivity.ConnectorVersion, error) {
		if !hasTwoSecondDeadline(ctx) {
			t.Fatal("set completion resolver context must have a two-second deadline")
		}
		resolvedClient = got
		return versionWithFullSchema(), nil
	}
	completion := testCompleteSetValues(func(cmd *cobra.Command) (*sdk.FormanceV1, error) {
		if !hasTwoSecondDeadline(cmd.Context()) {
			t.Fatal("set completion factory context is interactive")
		}
		return client, nil
	}, resolveVersion, mockPathCompleter(nil))
	command := &cobra.Command{}
	command.Flags().StringArray("set", nil, "")
	if err := command.Flags().Set("set", "TOKEN=secret://connector/token"); err != nil {
		t.Fatal(err)
	}

	candidates, directive := completion(command, nil, "")

	if resolvedClient != client {
		t.Fatal("resolver did not receive the factory client")
	}
	want := []string{
		"/etc/plugin/config.yaml=\trequired · Connector configuration",
		"API_URL=\trequired · API endpoint",
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
	completion := testCompleteSetValues(nil, nil, paths)

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
		listInstances: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
			return nil, errors.New("unsupported deployment")
		},
		connectorVersions: connectorVersions{
			listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*testConnectorVersionList, error) {
				return nil, errors.New("unsupported deployment")
			},
		},
	}
	tests := map[string]struct {
		completion cobra.CompletionFunc
		command    *cobra.Command
		args       []string
		prefix     string
	}{
		"instance authentication": {
			completion: testCompleteConnectionNames(func(*cobra.Command) (*sdk.FormanceV1, error) { return nil, errors.New("not authenticated") }),
			command:    &cobra.Command{},
		},
		"instance nil client": {
			completion: testCompleteConnectionNames(func(*cobra.Command) (*sdk.FormanceV1, error) { return nil, nil }),
			command:    &cobra.Command{},
		},
		"instance API": {completion: testCompleteConnectionNames(factoryReturning(apiErrorClient)), command: &cobra.Command{}},
		"instance pagination": {
			completion: testCompleteConnectionNames(factoryReturning(completionClientMock{listInstances: func(_ context.Context, options connectivityinternal.ListOptions) (*testConnectionList, error) {
				if options.Cursor != "" {
					return nil, errors.New("second page unavailable")
				}
				return &testConnectionList{HasMore: true, Next: "page-two"}, nil
			}})),
			command: &cobra.Command{},
		},
		"instance timeout": {
			completion: testCompleteConnectionNames(factoryReturning(completionClientMock{listInstances: func(ctx context.Context, _ connectivityinternal.ListOptions) (*testConnectionList, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}})),
			command: commandWithExpiredContext(),
		},
		"version empty connector": {completion: testCompleteVersions(factoryReturning(apiErrorClient), func(*cobra.Command, []string) string { return "" }), command: &cobra.Command{}},
		"version API":             {completion: testCompleteVersions(factoryReturning(apiErrorClient), func(*cobra.Command, []string) string { return "stripe" }), command: &cobra.Command{}},
		"set resolver": {
			completion: testCompleteSetValues(factoryReturning(completionClientMock{}), func(context.Context, *sdk.FormanceV1, *cobra.Command, []string) (*connectivity.ConnectorVersion, error) {
				return nil, errors.New("cannot resolve connector version")
			}, mockPathCompleter(nil)),
			command: &cobra.Command{},
		},
		"path": {
			completion: testCompleteSetValues(nil, nil, func(string) ([]string, error) { return nil, errors.New("cannot read directory") }),
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
	completion := testCompleteConnectionNames(func(cmd *cobra.Command) (*sdk.FormanceV1, error) {
		if !hasTwoSecondDeadline(cmd.Context()) {
			t.Fatal("copied completion command must have a two-second deadline")
		}
		return mockSDK(&completionClientMock{listInstances: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
			return &testConnectionList{}, nil
		}}), nil
	})

	_, _ = completion(command, nil, "")

	if command.Context() != originalContext {
		t.Fatal("completion mutated the original command context")
	}
	if hasTwoSecondDeadline(command.Context()) {
		t.Fatal("original command context acquired a completion deadline")
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
