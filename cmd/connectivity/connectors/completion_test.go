package connectors

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

func TestCompleteConnectorNamesUsesBoundedCatalogQueryAndReturnsPrefixMatchesWithDescriptions(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	var remaining time.Duration
	client := connectorClientMock{list: func(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
		gotOptions = options
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("completion context has no deadline")
		}
		remaining = time.Until(deadline)
		return &connectivityclient.ConnectorList{Items: []connectivityclient.Connector{
			connectorFixture("alpha"), connectorFixture("beta"),
		}}, nil
	}}

	completion := CompleteConnectorNames(func(cmd *cobra.Command) (connectivityclient.Client, error) {
		if !connectivityinternal.IsNonInteractive(cmd.Context()) {
			t.Fatal("completion factory context is interactive")
		}
		return client, nil
	})
	candidates, directive := completion(&cobra.Command{}, nil, "al")

	if !reflect.DeepEqual(gotOptions, connectivityclient.ListOptions{Limit: 500}) {
		t.Fatalf("ListConnectors options = %#v, want limit 500", gotOptions)
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

func TestCompleteConnectorNamesReturnsSilentlyOnFactoryAPIAndTimeoutErrors(t *testing.T) {
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
			factory: factoryReturning(connectorClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
				return nil, errors.New("unsupported deployment")
			}}),
		},
		"timeout": {
			command: commandWithExpiredContext(),
			factory: factoryReturning(connectorClientMock{list: func(ctx context.Context, _ connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
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
