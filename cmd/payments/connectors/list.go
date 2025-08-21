package connectors

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ConnectorData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type PaymentsConnectorsListStore struct {
	Connectors []ConnectorData `json:"connectors"`
}
type PaymentsConnectorsListController struct {
	PaymentsVersion versions.Version

	store *PaymentsConnectorsListStore

	pageSizeFlag string
}

func (c *PaymentsConnectorsListController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*PaymentsConnectorsListStore] = (*PaymentsConnectorsListController)(nil)

func NewDefaultPaymentsConnectorsListStore() *PaymentsConnectorsListStore {
	return &PaymentsConnectorsListStore{
		Connectors: []ConnectorData{},
	}
}

func NewPaymentsConnectorsListController() *PaymentsConnectorsListController {
	return &PaymentsConnectorsListController{
		store:        NewDefaultPaymentsConnectorsListStore(),
		pageSizeFlag: "page-size",
	}
}

func NewListCommand() *cobra.Command {
	c := NewPaymentsConnectorsListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List all enabled connectors"),
		fctl.WithIntFlag(c.pageSizeFlag, 10, "Page size"),
		fctl.WithController[*PaymentsConnectorsListStore](c),
	)
}

func (c *PaymentsConnectorsListController) GetStore() *PaymentsConnectorsListStore {
	return c.store
}

func (c *PaymentsConnectorsListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	pageSizeAsInt := int64(fctl.GetInt(cmd, c.pageSizeFlag))

	switch c.PaymentsVersion {
	case versions.V3:
		response, err := store.Client().Payments.V3.ListConnectors(cmd.Context(), operations.V3ListConnectorsRequest{
			PageSize: &pageSizeAsInt,
		})
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		if response.V3ConnectorsCursorResponse == nil {
			return nil, fmt.Errorf("unexpected response: %v", response)
		}
		c.store.Connectors = fctl.Map(response.V3ConnectorsCursorResponse.Cursor.Data, V3toConnectorData)

	case versions.V0, versions.V1, versions.V2:
		response, err := store.Client().Payments.V1.ListAllConnectors(cmd.Context())
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		if response.ConnectorsResponse == nil {
			return nil, fmt.Errorf("unexpected response: %v", response)
		}

		connectorsLength := len(response.ConnectorsResponse.Data)
		endIndex := int(pageSizeAsInt)
		if connectorsLength < endIndex {
			endIndex = connectorsLength
		}

		connectorsSlice := response.ConnectorsResponse.Data[:endIndex]
		c.store.Connectors = fctl.Map(connectorsSlice, V1toConnectorData)

	}

	return c, nil
}

func (c *PaymentsConnectorsListController) Render(cmd *cobra.Command, args []string) error {
	tableData := fctl.Map(c.store.Connectors, func(connector ConnectorData) []string {
		if c.PaymentsVersion >= versions.V1 {
			return []string{
				connector.Provider,
				connector.Name,
				connector.ID,
			}
		} else {
			// V0
			return []string{
				connector.Provider,
			}
		}

	})
	if c.PaymentsVersion >= versions.V1 {
		tableData = fctl.Prepend(tableData, []string{"Provider", "Name", "ConnectorID"})
	} else {
		tableData = fctl.Prepend(tableData, []string{"Provider"})
	}

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}

func V3toConnectorData(connector shared.V3Connector) ConnectorData {
	return ConnectorData{
		ID:       connector.ID,
		Name:     connector.Name,
		Provider: connector.Provider,
	}
}

func V1toConnectorData(connector shared.ConnectorsResponseData) ConnectorData {
	return ConnectorData{
		ID:       connector.ConnectorID,
		Name:     connector.Name,
		Provider: string(connector.Provider),
	}
}
