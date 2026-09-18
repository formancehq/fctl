package connectors

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type FacetsStore struct {
	Total  int64                       `json:"total"`
	Facets map[string]map[string]int64 `json:"facets"`
}

type FacetsController struct {
	factory connectivityinternal.ClientFactory
	store   *FacetsStore
}

var _ fctl.Controller[*FacetsStore] = (*FacetsController)(nil)

func NewFacetsController(factory connectivityinternal.ClientFactory) *FacetsController {
	return &FacetsController{
		factory: factory,
		store:   &FacetsStore{Facets: map[string]map[string]int64{}},
	}
}

func NewFacetsCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewFacetsController(factory)
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
		connectivityinternal.CompleteFilterExpressions(factory, connectivityclient.ResourceConnectors),
	); err != nil {
		panic(err)
	}
	return command
}

func (c *FacetsController) GetStore() *FacetsStore {
	return c.store
}

func (c *FacetsController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}
	query, err := connectivityinternal.GetListQuery(cmd)
	if err != nil {
		return nil, err
	}

	response, err := client.GetConnectorFacets(cmd.Context(), query)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("get connectivity connector facets: empty response")
	}

	c.store.Total = response.Total
	c.store.Facets = response.Facets
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
