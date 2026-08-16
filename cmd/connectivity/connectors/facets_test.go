package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func facetDistributionFixture() *connectivityclient.FacetDistribution {
	return &connectivityclient.FacetDistribution{Total: 8, Facets: map[string]map[string]int64{
		"provider": {"psp": 6, "bank": 2},
		"domain":   {"payouts": 3},
	}}
}

func TestFacetsRendersTotalAndSortedDistribution(t *testing.T) {
	var gotQuery string
	client := connectorClientMock{facets: func(_ context.Context, query string) (*connectivityclient.FacetDistribution, error) {
		gotQuery = query
		return facetDistributionFixture(), nil
	}}

	output, err := executeCommand(NewFacetsCommand(factoryReturning(client)), "--filter", "catalog=ee")
	if err != nil {
		t.Fatalf("execute facets command: %v", err)
	}

	if want := `{"$match":{"catalog":"ee"}}`; gotQuery != want {
		t.Fatalf("GetConnectorFacets query = %q, want %q", gotQuery, want)
	}
	for _, expected := range []string{
		"Total", "8",
		"Facet", "Value", "Connectors",
		"domain", "payouts", "3",
		"provider", "bank", "2",
		"psp", "6",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("plain output missing %q:\n%s", expected, output)
		}
	}
	if strings.Index(output, "domain") > strings.Index(output, "provider") {
		t.Errorf("facets are not sorted by facet name:\n%s", output)
	}
}

func TestFacetsJSONPreservesCompleteDistribution(t *testing.T) {
	client := connectorClientMock{facets: func(context.Context, string) (*connectivityclient.FacetDistribution, error) {
		return facetDistributionFixture(), nil
	}}

	command := NewFacetsCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err := executeCommand(command, "--output", "json")
	if err != nil {
		t.Fatalf("execute JSON facets command: %v", err)
	}

	var envelope struct {
		Data FacetsStore `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}
	if envelope.Data.Total != 8 || !reflect.DeepEqual(envelope.Data.Facets, facetDistributionFixture().Facets) {
		t.Fatalf("JSON facets = %#v, want the complete distribution", envelope.Data)
	}
}

func TestFacetsReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (connectivityclient.Client, error)
		want    string
	}{
		"factory": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return nil, errors.New("authentication failed")
			},
			want: "authentication failed",
		},
		"API": {
			factory: factoryReturning(connectorClientMock{facets: func(context.Context, string) (*connectivityclient.FacetDistribution, error) {
				return nil, errors.New("catalog unavailable")
			}}),
			want: "catalog unavailable",
		},
		"empty response": {
			factory: factoryReturning(connectorClientMock{facets: func(context.Context, string) (*connectivityclient.FacetDistribution, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(NewFacetsCommand(test.factory))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}
