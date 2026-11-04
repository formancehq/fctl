package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func facetDistributionFixture() *connectivity.FacetDistribution {
	return &connectivity.FacetDistribution{Total: 8, Facets: map[string]map[string]int64{
		"provider": {"psp": 6, "bank": 2},
		"domain":   {"payouts": 3},
	}}
}

func TestFacetsRendersTotalAndSortedDistribution(t *testing.T) {
	var gotQuery string
	client := connectorTransportMock{facets: func(_ context.Context, query string) (*connectivity.FacetDistribution, error) {
		gotQuery = query
		return facetDistributionFixture(), nil
	}}

	output, err := executeCommand(testNewFacetsCommand(sdkFor(client)), "--filter", "catalog=ee")
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
	client := connectorTransportMock{facets: func(context.Context, string) (*connectivity.FacetDistribution, error) {
		return facetDistributionFixture(), nil
	}}

	command := testNewFacetsCommand(sdkFor(client))
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

func TestFacetsReturnsTransportAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		client *formance.FormanceV1
		want   string
	}{
		"API": {
			client: sdkFor(connectorTransportMock{facets: func(context.Context, string) (*connectivity.FacetDistribution, error) {
				return nil, errors.New("catalog unavailable")
			}}),
			want: "catalog unavailable",
		},
		"empty response": {
			client: sdkFor(connectorTransportMock{facets: func(context.Context, string) (*connectivity.FacetDistribution, error) {
				return nil, nil
			}}),
			want: "unexpected end of JSON input",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(testNewFacetsCommand(test.client))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}
