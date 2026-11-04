package connectors

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func TestCompleteConnectorNamesUsesBoundedCatalogQueryAndReturnsPrefixMatchesWithDescriptions(t *testing.T) {
	var gotOptions connectivityinternal.ListOptions
	var remaining time.Duration
	client := connectorClientMock{list: func(ctx context.Context, options connectivityinternal.ListOptions) (*connectorPage, error) {
		gotOptions = options
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("completion context has no deadline")
		}
		remaining = time.Until(deadline)
		return &connectorPage{Items: []connectivity.Connector{
			connectorFixture("alpha"), connectorFixture("beta"),
		}}, nil
	}}

	completion := CompleteConnectorNames(func(cmd *cobra.Command) (connectivityinternal.API, error) {
		if !connectivityinternal.IsNonInteractive(cmd.Context()) {
			t.Fatal("completion factory context is interactive")
		}
		return client, nil
	})
	candidates, directive := completion(&cobra.Command{}, nil, "al")

	if !reflect.DeepEqual(gotOptions, connectivityinternal.ListOptions{PageSize: 100}) {
		t.Fatalf("ListConnectors options = %#v, want page size 100", gotOptions)
	}
	if remaining <= 1500*time.Millisecond || remaining > 2*time.Second {
		t.Fatalf("completion deadline remaining = %s, want approximately 2s", remaining)
	}
	if !reflect.DeepEqual(candidates, []string{"alpha\tConnector description"}) {
		t.Fatalf("candidates = %#v, want prefix match with description", candidates)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteConnectorNamesFollowsCursorPagination(t *testing.T) {
	var gotCursors []string
	client := connectorClientMock{list: func(_ context.Context, options connectivityinternal.ListOptions) (*connectorPage, error) {
		gotCursors = append(gotCursors, options.Cursor)
		if options.Cursor == "" {
			return &connectorPage{
				Items:   []connectivity.Connector{connectorFixture("alpha")},
				HasMore: true,
				Next:    "page-two",
			}, nil
		}
		return &connectorPage{Items: []connectivity.Connector{connectorFixture("alpine")}}, nil
	}}

	candidates, _ := CompleteConnectorNames(factoryReturning(client))(&cobra.Command{}, nil, "al")

	if !reflect.DeepEqual(gotCursors, []string{"", "page-two"}) {
		t.Fatalf("list cursors = %#v, want the continuation to be followed", gotCursors)
	}
	want := []string{"alpha\tConnector description", "alpine\tConnector description"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want both pages merged %#v", candidates, want)
	}
}

func TestCompleteConnectorNamesReturnsSilentlyOnFactoryAPIAndTimeoutErrors(t *testing.T) {
	tests := map[string]struct {
		command *cobra.Command
		factory func(*cobra.Command) (connectivityinternal.API, error)
	}{
		"authentication": {
			command: &cobra.Command{},
			factory: func(*cobra.Command) (connectivityinternal.API, error) {
				return nil, errors.New("not authenticated")
			},
		},
		"API": {
			command: &cobra.Command{},
			factory: factoryReturning(connectorClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*connectorPage, error) {
				return nil, errors.New("unsupported deployment")
			}}),
		},
		"timeout": {
			command: commandWithExpiredContext(),
			factory: factoryReturning(connectorClientMock{list: func(ctx context.Context, _ connectivityinternal.ListOptions) (*connectorPage, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			candidates, directive := CompleteConnectorNames(test.factory)(test.command, nil, "")
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

func TestCompleteConnectorNamesUsesFlatNameAndOptionalDescription(t *testing.T) {
	client := connectorClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*connectorPage, error) {
		return &connectorPage{Items: []connectivity.Connector{
			{Name: "stripe"}, {Name: ""}, {Name: "wise"},
		}}, nil
	}}
	candidates, directive := CompleteConnectorNames(factoryReturning(client))(&cobra.Command{}, nil, "")
	if !reflect.DeepEqual(candidates, []string{"stripe", "wise"}) {
		t.Fatalf("flat connector candidates = %#v", candidates)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}
