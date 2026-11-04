package connectors

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type FacetsStore struct {
	Total  int64                       `json:"total"`
	Facets map[string]map[string]int64 `json:"facets"`
}

type FacetsController struct {
	store *FacetsStore
}

var _ fctl.Controller[*FacetsStore] = (*FacetsController)(nil)

func NewFacetsController() *FacetsController {
	return &FacetsController{
		store: &FacetsStore{Facets: map[string]map[string]int64{}},
	}
}

func NewFacetsCommand() *cobra.Command {
	controller := NewFacetsController()
	command := fctl.NewCommand(
		"facets",
		fctl.WithAliases("facet", "f"),
		fctl.WithShortDescription("Show the facet-value distribution of the connector catalogue"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		connectivityinternal.WithListQueryFlags(),
		fctl.WithController[*FacetsStore](controller),
	)
	if err := command.RegisterFlagCompletionFunc(
		connectivityinternal.FilterFlag,
		connectivityinternal.CompleteFilterExpressions(connectivityinternal.ResourceConnectors),
	); err != nil {
		panic(err)
	}
	return command
}

func (c *FacetsController) GetStore() *FacetsStore {
	return c.store
}

func (c *FacetsController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, nil, stackClient.Connectivity.V1)
}

func (c *FacetsController) run(cmd *cobra.Command, _ []string, client *formance.FormanceV1) (fctl.Renderable, error) {
	query, err := connectivityinternal.GetListQuery(cmd)
	if err != nil {
		return nil, err
	}

	response, err := client.GetConnectorFacets(cmd.Context(), operations.GetConnectorFacetsRequest{Query: connectivityinternal.OptionalString(query)}, nil)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("get connectivity connector facets: empty response")
	}

	if err := connectivityinternal.ResponseError(response.StatusCode, response.Error, response.RawResponse); err != nil {
		return nil, err
	}
	if response.FacetDistribution == nil {
		return nil, fmt.Errorf("get connectivity connector facets: empty response")
	}
	if response.FacetDistribution.Facets == nil {
		return nil, fmt.Errorf("get connectivity connector facets: facets must be an object")
	}
	c.store.Total = response.FacetDistribution.Total
	c.store.Facets = response.FacetDistribution.Facets
	return c, nil
}

func (c *FacetsController) Render(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "Total: %d\n", c.store.Total); err != nil {
		return err
	}

	rows := make([][]string, 0)
	facets := make([]string, 0, len(c.store.Facets))
	for facet := range c.store.Facets {
		facets = append(facets, facet)
	}
	sort.Strings(facets)
	for _, facet := range facets {
		values := make([]string, 0, len(c.store.Facets[facet]))
		for value := range c.store.Facets[facet] {
			values = append(values, value)
		}
		sort.Strings(values)
		for _, value := range values {
			rows = append(rows, []string{facet, value, strconv.FormatInt(c.store.Facets[facet][value], 10)})
		}
	}
	rows = fctl.Prepend(rows, []string{"Facet", "Value", "Connectors"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(rows).
		Render()
}
