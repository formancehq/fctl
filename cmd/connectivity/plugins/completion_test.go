package plugins

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

func TestCompletePluginNamesUsesBoundedCatalogQueryAndReturnsPrefixMatchesWithDescriptions(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	var remaining time.Duration
	client := pluginClientMock{list: func(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
		gotOptions = options
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("completion context has no deadline")
		}
		remaining = time.Until(deadline)
		return &connectivityclient.PluginList{Items: []connectivityclient.Plugin{
			pluginFixture("alpha"), pluginFixture("beta"),
		}}, nil
	}}

	completion := CompletePluginNames(factoryReturning(client))
	candidates, directive := completion(&cobra.Command{}, nil, "al")

	if !reflect.DeepEqual(gotOptions, connectivityclient.ListOptions{Limit: 500}) {
		t.Fatalf("ListPlugins options = %#v, want limit 500", gotOptions)
	}
	if remaining <= 1500*time.Millisecond || remaining > 2*time.Second {
		t.Fatalf("completion deadline remaining = %s, want approximately 2s", remaining)
	}
	if !reflect.DeepEqual(candidates, []string{"alpha\tPlugin description"}) {
		t.Fatalf("candidates = %#v, want prefix match with description", candidates)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompletePluginNamesReturnsSilentlyOnFactoryAPIAndTimeoutErrors(t *testing.T) {
	tests := map[string]struct {
		command *cobra.Command
		factory func(*cobra.Command) (connectivityclient.Client, error)
	}{
		"authentication": {
			command: &cobra.Command{},
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return nil, errors.New("not authenticated")
			},
		},
		"API": {
			command: &cobra.Command{},
			factory: factoryReturning(pluginClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
				return nil, errors.New("unsupported deployment")
			}}),
		},
		"timeout": {
			command: commandWithExpiredContext(),
			factory: factoryReturning(pluginClientMock{list: func(ctx context.Context, _ connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			candidates, directive := CompletePluginNames(test.factory)(test.command, nil, "")
			if len(candidates) != 0 {
				t.Fatalf("candidates = %#v, want none", candidates)
			}
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("directive = %v, want NoFileComp", directive)
			}
		})
	}
}

func commandWithExpiredContext() *cobra.Command {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := &cobra.Command{}
	command.SetContext(ctx)
	return command
}
