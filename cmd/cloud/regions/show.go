package regions

import (
	"fmt"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	Region components.AnyRegion `json:"region"`
}
type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewDefaultShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("show <region-id>",
		fctl.WithAliases("sh", "s"),
		fctl.WithShortDescription("Show region details"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*ShowStore](NewShowController()),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	request := operations.GetRegionRequest{
		OrganizationID: organizationID,
		RegionID:       args[0],
	}

	response, err := apiClient.GetRegion(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.GetRegionResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Region = response.GetRegionResponse.GetData()

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) (err error) {
	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Information")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.Region.GetID()})
	tableData = append(tableData, []string{pterm.LightCyan("Name"), c.store.Region.GetName()})
	tableData = append(tableData, []string{pterm.LightCyan("Base URL"), c.store.Region.GetBaseURL()})
	tableData = append(tableData, []string{pterm.LightCyan("Active"), fctl.BoolToString(c.store.Region.GetActive())})
	tableData = append(tableData, []string{pterm.LightCyan("Public"), fctl.BoolToString(c.store.Region.GetPublic())})

	if version := c.store.Region.GetVersion(); version != nil {
		tableData = append(tableData, []string{pterm.LightCyan("Version"), *version})
	}

	if creator := c.store.Region.GetCreator(); creator != nil {
		tableData = append(tableData, []string{pterm.LightCyan("Creator"), creator.GetEmail()})
	}
	if lastPing := c.store.Region.GetLastPing(); lastPing != nil {
		tableData = append(tableData, []string{pterm.LightCyan("Last ping"), lastPing.Format(time.RFC3339)})
	}

	err = pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
	if err != nil {
		return
	}

	tableData = pterm.TableData{}
	capabilities, err := fctl.StructToMap(c.store.Region.GetCapabilities())
	if err != nil {
		return
	}
	if len(capabilities) > 0 {
		fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Capabilities")
	}
	for key, value := range capabilities {
		data := []string{
			pterm.LightCyan(key),
		}

		var v []string
		if value != nil {
			c, ok := value.([]interface{})
			if ok {
				for _, val := range c {
					v = append(v, fmt.Sprintf("%v", val))
				}
			}
		}
		data = append(data, strings.Join(v, ", "))
		tableData = append(tableData, data)
	}

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
