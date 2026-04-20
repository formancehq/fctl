package deployments

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/go-libs/v4/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type List struct {
	components.ListDeploymentsResponseData
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ListDeploymentsResponseData: components.ListDeploymentsResponseData{},
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
		fctl.WithShortDescription("List deployments"),
		fctl.WithStringFlag("app-id", "", "Filter by app ID"),
		fctl.WithIntFlag("page", 1, "Page number"),
		fctl.WithIntFlag("page-size", 100, "Page size"),
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

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}

	var appID *string
	if id := fctl.GetString(cmd, "app-id"); id != "" {
		appID = &id
	}

	deployments, err := apiClient.ListDeployments(
		cmd.Context(),
		appID,
		pointer.For(int64(fctl.GetInt(cmd, "page"))),
		pointer.For(int64(fctl.GetInt(cmd, "page-size"))),
	)
	if err != nil {
		return nil, err
	}

	c.store.ListDeploymentsResponseData = deployments.ListDeploymentsResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, _ []string) error {
	data := [][]string{
		{"ID", "App ID", "Run Status", "Manifest ID", "Created At"},
	}

	for _, d := range c.store.Items {
		data = append(data, []string{
			d.ID,
			d.AppID,
			d.RunStatus,
			func() string {
				if d.ManifestID != nil {
					return *d.ManifestID
				}
				return "inline"
			}(),
			fmt.Sprint(d.CreatedAt),
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
