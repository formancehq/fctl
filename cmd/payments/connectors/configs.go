package connectors

import (
	"fmt"
	"sort"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	formance "github.com/formancehq/formance-sdk-go/v4"

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

func NewConnectorListAvailableCommand() *cobra.Command {
	c := NewConnectorConfigsController()
	return fctl.NewCommand("list-available",
		fctl.WithAliases("la"),
		fctl.WithShortDescription("List connectors available for install (dependent on Connectivity module version) and their configuration schemas"),
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
		if err := c.legacyV1Configs(cmd, stackClient); err != nil {
			return nil, err
		}
		return c, nil
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
			c.store.Fields = append(c.store.Fields, ConnectorConfigField{
				Connector:    connectorName,
				Field:        fieldName,
				DataType:     meta.DataType,
				Required:     meta.Required,
				DefaultValue: def,
			})
		}
	}

	return c, nil
}

func (c *ConnectorConfigsController) legacyV1Configs(cmd *cobra.Command, stackClient *formance.Formance) error {
	response, err := stackClient.Payments.V1.ListConfigsAvailableConnectors(cmd.Context())
	if err != nil {
		return err
	}
	if response.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	if response.ConnectorsConfigsResponse == nil {
		return fmt.Errorf("unexpected empty response")
	}
	data := response.ConnectorsConfigsResponse.Data
	connectorNames := make([]string, 0, len(data))
	for name := range data {
		connectorNames = append(connectorNames, name)
	}
	sort.Strings(connectorNames)
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
			c.store.Fields = append(c.store.Fields, ConnectorConfigField{
				Connector:    connectorName,
				Field:        fieldName,
				DataType:     meta.DataType,
				Required:     meta.Required,
				DefaultValue: def,
			})
		}
	}
	return nil
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
