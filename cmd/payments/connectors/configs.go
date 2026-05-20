package connectors

import (
	"fmt"
	"sort"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ConnectorConfigField struct {
	Connector    string `json:"connector"`
	Field        string `json:"field"`
	DataType     string `json:"dataType"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

type ConnectorConfigsStore struct {
	Fields []ConnectorConfigField `json:"fields"`
}

type ConnectorConfigsController struct {
	PaymentsVersion versions.Version
	store           *ConnectorConfigsStore
}

func (c *ConnectorConfigsController) SetVersion(v versions.Version) {
	c.PaymentsVersion = v
}

var _ fctl.Controller[*ConnectorConfigsStore] = (*ConnectorConfigsController)(nil)

func NewConnectorConfigsController() *ConnectorConfigsController {
	return &ConnectorConfigsController{
		store: &ConnectorConfigsStore{},
	}
}

func NewConnectorConfigsCommand() *cobra.Command {
	c := NewConnectorConfigsController()
	return fctl.NewCommand("configs",
		fctl.WithAliases("cf"),
		fctl.WithShortDescription("List available connectors and their configuration schemas"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithController[*ConnectorConfigsStore](c),
	)
}

func (c *ConnectorConfigsController) GetStore() *ConnectorConfigsStore {
	return c.store
}

func (c *ConnectorConfigsController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion.Major < versions.V3 {
		return nil, fmt.Errorf("connector configs discovery requires payments API v3 or later")
	}

	response, err := stackClient.Payments.V3.ListConnectorConfigs(cmd.Context())
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	if response.V3ConnectorConfigsResponse == nil {
		return nil, fmt.Errorf("unexpected empty response")
	}

	data := response.V3ConnectorConfigsResponse.Data

	connectorNames := make([]string, 0, len(data))
	for name := range data {
		connectorNames = append(connectorNames, name)
	}
	sort.Strings(connectorNames)

	var fields []ConnectorConfigField
	for _, connectorName := range connectorNames {
		fieldMap := data[connectorName]
		fieldNames := make([]string, 0, len(fieldMap))
		for f := range fieldMap {
			fieldNames = append(fieldNames, f)
		}
		sort.Strings(fieldNames)

		for _, fieldName := range fieldNames {
			meta := fieldMap[fieldName]
			def := ""
			if meta.DefaultValue != nil {
				def = *meta.DefaultValue
			}
			fields = append(fields, ConnectorConfigField{
				Connector:    connectorName,
				Field:        fieldName,
				DataType:     meta.DataType,
				Required:     meta.Required,
				DefaultValue: def,
			})
		}
	}

	c.store.Fields = fields
	return c, nil
}

func (c *ConnectorConfigsController) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{{"Connector", "Field", "Type", "Required", "Default"}}

	lastConnector := ""
	for _, f := range c.store.Fields {
		connectorLabel := f.Connector
		if connectorLabel == lastConnector {
			connectorLabel = ""
		} else {
			lastConnector = f.Connector
		}
		req := "no"
		if f.Required {
			req = "yes"
		}
		tableData = append(tableData, []string{
			pterm.LightCyan(connectorLabel),
			f.Field,
			f.DataType,
			req,
			f.DefaultValue,
		})
	}

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
