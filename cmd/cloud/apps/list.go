package apps

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/go-libs/v3/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type List struct {
	components.ListAppsResponseData
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ListAppsResponseData: components.ListAppsResponseData{},
	}
}

func NewListCtrl() *ListCtrl {
	return &ListCtrl{
		store: newDefaultStore(),
	}
}

func NewList() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls"),
		fctl.WithIntFlag("page", 1, "Page number"),
		fctl.WithIntFlag("page-size", 100, "Page size"),
		fctl.WithShortDescription("List apps"),
		fctl.WithController(NewListCtrl()),
	)
}

func (c *ListCtrl) GetStore() *List {
	return c.store
}

func (c *ListCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	pageSize := fctl.GetInt(cmd, "page-size")
	page := fctl.GetInt(cmd, "page")

	organizationID, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}
	apps, err := apiClient.ListApps(
		cmd.Context(),
		organizationID,
		pointer.For(int64(page)),
		pointer.For(int64(pageSize)),
	)
	if err != nil {
		return nil, err
	}

	c.store.ListAppsResponseData = apps.ListAppsResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, _ []string) error {
	data := [][]string{
		{"Name", "ID", "Run Status", "Has Configuration Version"},
	}

	for _, w := range c.store.Items {
		data = append(data, []string{
			w.Name,
			w.ID,
			func() string {
				if w.CurrentRun == nil {
					return "N/A"
				}
				return w.CurrentRun.Status
			}(),
			func() string {
				if w.CurrentConfigurationVersion != nil {
					return "Yes"
				}
				return "No"
			}(),
		})
	}

	if err := pterm.
		DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(data).
		Render(); err != nil {
		return err
	}
	return nil
}
