package internal

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

type filterCompletionClientMock struct {
	connectivityclient.Client
	capabilities   func(context.Context) (*connectivityclient.QueryCapabilities, error)
	facets         func(context.Context, string) (*connectivityclient.FacetDistribution, error)
	listConnectors func(context.Context, connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error)
}

func (m filterCompletionClientMock) GetQueryCapabilities(ctx context.Context) (*connectivityclient.QueryCapabilities, error) {
	return m.capabilities(ctx)
}

func (m filterCompletionClientMock) GetConnectorFacets(ctx context.Context, query string) (*connectivityclient.FacetDistribution, error) {
	return m.facets(ctx, query)
}

func (m filterCompletionClientMock) ListConnectors(ctx context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
	return m.listConnectors(ctx, options)
}

func capabilitiesFixture() *connectivityclient.QueryCapabilities {
	return &connectivityclient.QueryCapabilities{Resources: map[string]map[string]connectivityclient.QueryFieldCapability{
		connectivityclient.ResourceConnectors: {
			"catalog": {Operators: []string{"$match", "$in", "$exists"}},
			"name":    {Operators: []string{"$match", "$in", "$like"}},
			"tags":    {Operators: []string{"$match", "$in", "$exists"}},
		},
		connectivityclient.ResourceConnectorInstances: {
			"channel":   {Operators: []string{"$match", "$in", "$exists"}, Enum: []string{"stable", "rc", "beta", "alpha"}},
			"connector": {Operators: []string{"$match", "$in"}},
		},
	}}
}

func filterFactory(t *testing.T, client connectivityclient.Client) ClientFactory {
	t.Helper()
	return func(cmd *cobra.Command) (connectivityclient.Client, error) {
		if !IsNonInteractive(cmd.Context()) {
			t.Fatal("filter completion factory context is interactive")
		}
		if _, ok := cmd.Context().Deadline(); !ok {
			t.Fatal("filter completion context has no deadline")
		}
		return client, nil
	}
}

func TestCompleteFilterExpressionsOffersCapabilityKeysWithOperators(t *testing.T) {
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteFilterExpressions(filterFactory(t, client), connectivityclient.ResourceConnectors)

	candidates, directive := completion(&cobra.Command{}, nil, "")

	want := []string{
		"catalog=\toperators: $match, $in, $exists",
		"name=\toperators: $match, $in, $like",
		"tags=\toperators: $match, $in, $exists",
	}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace {
		t.Fatalf("directive = %v, want NoFileComp|NoSpace", directive)
	}
}

func TestCompleteFilterExpressionsFiltersKeysByPrefix(t *testing.T) {
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteFilterExpressions(filterFactory(t, client), connectivityclient.ResourceConnectors)

	candidates, _ := completion(&cobra.Command{}, nil, "na")

	want := []string{"name=\toperators: $match, $in, $like"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}

func TestCompleteFilterExpressionsCompletesEnumValuesForTypedOperator(t *testing.T) {
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteFilterExpressions(filterFactory(t, client), connectivityclient.ResourceConnectorInstances)

	equals, directive := completion(&cobra.Command{}, nil, "channel=st")
	negated, _ := completion(&cobra.Command{}, nil, "channel!=r")

	if want := []string{"channel=stable"}; !reflect.DeepEqual(equals, want) {
		t.Fatalf("equality candidates = %#v, want %#v", equals, want)
	}
	if want := []string{"channel!=rc"}; !reflect.DeepEqual(negated, want) {
		t.Fatalf("negation candidates = %#v, want %#v", negated, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteFilterExpressionsCompletesTagValuesFromFacetDistribution(t *testing.T) {
	client := filterCompletionClientMock{
		capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
			return capabilitiesFixture(), nil
		},
		facets: func(context.Context, string) (*connectivityclient.FacetDistribution, error) {
			return &connectivityclient.FacetDistribution{Total: 8, Facets: map[string]map[string]int64{
				"provider": {"psp": 6, "bank": 2},
				"domain":   {"payouts": 3},
			}}, nil
		},
	}
	completion := CompleteFilterExpressions(filterFactory(t, client), connectivityclient.ResourceConnectors)

	all, _ := completion(&cobra.Command{}, nil, "tags=")
	scoped, _ := completion(&cobra.Command{}, nil, "tags=provider:")

	wantAll := []string{
		"tags=domain:payouts\t3 connectors",
		"tags=provider:bank\t2 connectors",
		"tags=provider:psp\t6 connectors",
	}
	if !reflect.DeepEqual(all, wantAll) {
		t.Fatalf("candidates = %#v, want %#v", all, wantAll)
	}
	wantScoped := []string{
		"tags=provider:bank\t2 connectors",
		"tags=provider:psp\t6 connectors",
	}
	if !reflect.DeepEqual(scoped, wantScoped) {
		t.Fatalf("scoped candidates = %#v, want %#v", scoped, wantScoped)
	}
}

func TestCompleteFilterExpressionsCompletesConnectorNamesForInstanceConnectorKey(t *testing.T) {
	displayName := "Stripe"
	stripe := "stripe"
	wise := "wise"
	client := filterCompletionClientMock{
		capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
			return capabilitiesFixture(), nil
		},
		listConnectors: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.ConnectorList, error) {
			if options.PageSize != 100 {
				t.Fatalf("ListConnectors page size = %d, want 100", options.PageSize)
			}
			return &connectivityclient.ConnectorList{Items: []connectivityclient.Connector{
				{Metadata: connectivityclient.ObjectMeta{Name: &wise}},
				{Metadata: connectivityclient.ObjectMeta{Name: &stripe}, Spec: connectivityclient.ConnectorSpec{DisplayName: &displayName}},
			}}, nil
		},
	}
	completion := CompleteFilterExpressions(filterFactory(t, client), connectivityclient.ResourceConnectorInstances)

	candidates, _ := completion(&cobra.Command{}, nil, "connector=")

	want := []string{"connector=stripe\tStripe", "connector=wise"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}

func TestCompleteChannelsServesTheCapabilityEnum(t *testing.T) {
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteChannels(filterFactory(t, client))

	all, directive := completion(&cobra.Command{}, nil, "")
	prefixed, _ := completion(&cobra.Command{}, nil, "s")

	if want := []string{"alpha", "beta", "rc", "stable"}; !reflect.DeepEqual(all, want) {
		t.Fatalf("candidates = %#v, want %#v", all, want)
	}
	if want := []string{"stable"}; !reflect.DeepEqual(prefixed, want) {
		t.Fatalf("prefixed candidates = %#v, want %#v", prefixed, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteChannelsReturnsSilentlyWhenCapabilitiesFail(t *testing.T) {
	completion := CompleteChannels(func(*cobra.Command) (connectivityclient.Client, error) {
		return filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
			return nil, errors.New("unsupported deployment")
		}}, nil
	})

	candidates, directive := completion(&cobra.Command{}, nil, "")

	if len(candidates) != 0 {
		t.Fatalf("candidates = %#v, want none", candidates)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteFilterExpressionsReturnsSilentlyWhenLookupsFail(t *testing.T) {
	tests := map[string]struct {
		factory  ClientFactory
		resource string
		prefix   string
	}{
		"factory error": {
			factory:  func(*cobra.Command) (connectivityclient.Client, error) { return nil, errors.New("not authenticated") },
			resource: connectivityclient.ResourceConnectors,
		},
		"nil factory": {
			resource: connectivityclient.ResourceConnectors,
		},
		"capabilities error": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
					return nil, errors.New("unsupported deployment")
				}}, nil
			},
			resource: connectivityclient.ResourceConnectors,
		},
		"unknown resource": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
					return capabilitiesFixture(), nil
				}}, nil
			},
			resource: "unknown",
		},
		"value for unknown key": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return filterCompletionClientMock{capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
					return capabilitiesFixture(), nil
				}}, nil
			},
			resource: connectivityclient.ResourceConnectors,
			prefix:   "nope=",
		},
		"facets error": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return filterCompletionClientMock{
					capabilities: func(context.Context) (*connectivityclient.QueryCapabilities, error) {
						return capabilitiesFixture(), nil
					},
					facets: func(context.Context, string) (*connectivityclient.FacetDistribution, error) {
						return nil, errors.New("unsupported deployment")
					},
				}, nil
			},
			resource: connectivityclient.ResourceConnectors,
			prefix:   "tags=",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			completion := CompleteFilterExpressions(test.factory, test.resource)

			candidates, directive := completion(&cobra.Command{}, nil, test.prefix)

			if len(candidates) != 0 {
				t.Fatalf("candidates = %#v, want none", candidates)
			}
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("directive = %v, want NoFileComp", directive)
			}
		})
	}
}
