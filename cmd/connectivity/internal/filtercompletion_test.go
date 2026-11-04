package internal

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type filterCompletionClientMock struct {
	API
	capabilities   func(context.Context) (*connectivity.QueryCapabilities, error)
	facets         func(context.Context, string) (*connectivity.FacetDistribution, error)
	listConnectors func(context.Context, ListOptions) (*testConnectorList, error)
}

type testConnectorList struct {
	Items    []connectivity.Connector
	PageSize int32
	HasMore  bool
	Next     string
}

func (m filterCompletionClientMock) GetQueryCapabilities(ctx context.Context, _ *operations.GetQueryCapabilitiesSecurity, _ ...operations.Option) (*operations.GetQueryCapabilitiesResponse, error) {
	v, err := m.capabilities(ctx)
	if v == nil || err != nil {
		return nil, err
	}
	return &operations.GetQueryCapabilitiesResponse{StatusCode: 200, QueryCapabilities: v}, nil
}
func (m filterCompletionClientMock) GetConnectorFacets(ctx context.Context, req operations.GetConnectorFacetsRequest, _ *operations.GetConnectorFacetsSecurity, _ ...operations.Option) (*operations.GetConnectorFacetsResponse, error) {
	query := ""
	if req.Query != nil {
		query = *req.Query
	}
	v, err := m.facets(ctx, query)
	if v == nil || err != nil {
		return nil, err
	}
	return &operations.GetConnectorFacetsResponse{StatusCode: 200, FacetDistribution: v}, nil
}
func (m filterCompletionClientMock) ListConnectors(ctx context.Context, req operations.ListConnectorsRequest, _ *operations.ListConnectorsSecurity, _ ...operations.Option) (*operations.ListConnectorsResponse, error) {
	options := ListOptions{}
	if req.PageSize != nil {
		options.PageSize = int32(*req.PageSize)
	}
	if req.Cursor != nil {
		options.Cursor = *req.Cursor
	}
	if req.Query != nil {
		options.Query = *req.Query
	}
	v, err := m.listConnectors(ctx, options)
	if v == nil || err != nil {
		return nil, err
	}
	return &operations.ListConnectorsResponse{StatusCode: 200, ConnectorCursor: &connectivity.ConnectorCursor{Cursor: connectivity.ConnectorCursorCursor{Data: v.Items, PageSize: int(v.PageSize), HasMore: v.HasMore, Next: OptionalString(v.Next)}}}, nil
}

func capabilitiesFixture() *connectivity.QueryCapabilities {
	return &connectivity.QueryCapabilities{Resources: map[string]map[string]connectivity.QueryFieldCapability{
		ResourceConnectors: {
			"catalog": {Operators: []string{"$match", "$in", "$exists"}},
			"name":    {Operators: []string{"$match", "$in", "$like"}},
			"tags":    {Operators: []string{"$match", "$in", "$exists"}},
		},
		ResourceConnections: {
			"channel":        {Operators: []string{"$match", "$in", "$exists"}, Enum: []string{"stable", "rc", "beta", "alpha"}},
			"connector":      {Operators: []string{"$match", "$in"}},
			"lifecycle":      {Operators: []string{"$match", "$in", "$exists"}, Enum: []string{"pending", "active", "suspended", "completed", "failed", "unknown"}},
			"appliedVersion": {Operators: []string{"$match", "$in", "$exists", "$gt", "$gte", "$lt", "$lte"}},
		},
	}}
}

func filterFactory(t *testing.T, client API) ClientFactory {
	t.Helper()
	return func(cmd *cobra.Command) (API, error) {
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
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteFilterExpressions(filterFactory(t, client), ResourceConnectors)

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
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteFilterExpressions(filterFactory(t, client), ResourceConnectors)

	candidates, _ := completion(&cobra.Command{}, nil, "na")

	want := []string{"name=\toperators: $match, $in, $like"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}

func TestCompleteFilterExpressionsCompletesEnumValuesForTypedOperator(t *testing.T) {
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteFilterExpressions(filterFactory(t, client), ResourceConnections)

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
		capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
			return capabilitiesFixture(), nil
		},
		facets: func(context.Context, string) (*connectivity.FacetDistribution, error) {
			return &connectivity.FacetDistribution{Total: 8, Facets: map[string]map[string]int64{
				"provider": {"psp": 6, "bank": 2},
				"domain":   {"payouts": 3},
			}}, nil
		},
	}
	completion := CompleteFilterExpressions(filterFactory(t, client), ResourceConnectors)

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
		capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
			return capabilitiesFixture(), nil
		},
		listConnectors: func(_ context.Context, options ListOptions) (*testConnectorList, error) {
			if options.PageSize != 100 {
				t.Fatalf("ListConnectors page size = %d, want 100", options.PageSize)
			}
			return &testConnectorList{Items: []connectivity.Connector{
				{Name: wise},
				{Name: stripe, DisplayName: &displayName},
			}}, nil
		},
	}
	completion := CompleteFilterExpressions(filterFactory(t, client), ResourceConnections)

	candidates, _ := completion(&cobra.Command{}, nil, "connector=")

	want := []string{"connector=stripe\tStripe", "connector=wise"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}

func TestCompleteFilterExpressionsDrainsBoundedConnectorPages(t *testing.T) {
	first, second := "adyen", "stripe"
	client := filterCompletionClientMock{
		capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
			return capabilitiesFixture(), nil
		},
		listConnectors: func(_ context.Context, options ListOptions) (*testConnectorList, error) {
			switch options.Cursor {
			case "":
				return &testConnectorList{Items: []connectivity.Connector{{Name: first}}, HasMore: true, Next: "second"}, nil
			case "second":
				return &testConnectorList{Items: []connectivity.Connector{{Name: second}}}, nil
			default:
				t.Fatalf("unexpected connector cursor %q", options.Cursor)
				return nil, nil
			}
		},
	}

	candidates, _ := CompleteFilterExpressions(filterFactory(t, client), ResourceConnections)(&cobra.Command{}, nil, "connector=")

	if want := []string{"connector=adyen", "connector=stripe"}; !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}

func TestCompleteChannelsServesTheCapabilityEnum(t *testing.T) {
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
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
	completion := CompleteChannels(func(*cobra.Command) (API, error) {
		return filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
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
			factory:  func(*cobra.Command) (API, error) { return nil, errors.New("not authenticated") },
			resource: ResourceConnectors,
		},
		"nil factory": {
			resource: ResourceConnectors,
		},
		"capabilities error": {
			factory: func(*cobra.Command) (API, error) {
				return filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
					return nil, errors.New("unsupported deployment")
				}}, nil
			},
			resource: ResourceConnectors,
		},
		"unknown resource": {
			factory: func(*cobra.Command) (API, error) {
				return filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
					return capabilitiesFixture(), nil
				}}, nil
			},
			resource: "unknown",
		},
		"value for unknown key": {
			factory: func(*cobra.Command) (API, error) {
				return filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
					return capabilitiesFixture(), nil
				}}, nil
			},
			resource: ResourceConnectors,
			prefix:   "nope=",
		},
		"facets error": {
			factory: func(*cobra.Command) (API, error) {
				return filterCompletionClientMock{
					capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
						return capabilitiesFixture(), nil
					},
					facets: func(context.Context, string) (*connectivity.FacetDistribution, error) {
						return nil, errors.New("unsupported deployment")
					},
				}, nil
			},
			resource: ResourceConnectors,
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

func TestCompleteInstanceFiltersUsePublicCapabilityNamesWithoutAliases(t *testing.T) {
	client := filterCompletionClientMock{capabilities: func(context.Context) (*connectivity.QueryCapabilities, error) {
		return capabilitiesFixture(), nil
	}}
	completion := CompleteFilterExpressions(filterFactory(t, client), ResourceConnections)
	keys, _ := completion(&cobra.Command{}, nil, "")
	want := []string{
		"appliedVersion=\toperators: $match, $in, $exists, $gt, $gte, $lt, $lte",
		"channel=\toperators: $match, $in, $exists",
		"connector=\toperators: $match, $in",
		"lifecycle=\toperators: $match, $in, $exists",
	}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("instance filter keys = %#v, want public keys %#v", keys, want)
	}
	active, _ := completion(&cobra.Command{}, nil, "lifecycle=a")
	if !reflect.DeepEqual(active, []string{"lifecycle=active"}) {
		t.Fatalf("lifecycle candidates = %#v", active)
	}
	for _, alias := range []string{"phase", "phase=", "resolvedVersion", "resolvedVersion="} {
		candidates, _ := completion(&cobra.Command{}, nil, alias)
		if len(candidates) != 0 {
			t.Errorf("obsolete alias %q offered candidates %#v", alias, candidates)
		}
	}
}
