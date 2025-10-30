package runs

import (
	"fmt"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/pointer"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type List struct {
	components.ListRunsResponseData
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ListRunsResponseData: components.ListRunsResponseData{},
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
		fctl.WithShortDescription("List runs for an app"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithIntFlag("page", 1, "Page number"),
		fctl.WithIntFlag("page-size", 100, "Page size"),
		fctl.WithController(NewListCtrl()),
	)
}

func (c *ListCtrl) GetStore() *List {
	return c.store
}

func (c *ListCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	relyingParty, err := fctl.GetAuthRelyingParty(cmd.Context(), fctl.GetHttpClient(cmd), profile.MembershipURI)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewAppDeployClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		fctl.GetCurrentProfileName(cmd, *cfg),
		*profile,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	runs, err := apiClient.ReadAppRuns(cmd.Context(), id, pointer.For(int64(fctl.GetInt(cmd, "page"))), pointer.For(int64(fctl.GetInt(cmd, "page-size"))))
	if err != nil {
		return nil, err
	}

	c.store.ListRunsResponseData = runs.ListRunsResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Created At", "ID", "Configuration Id", "Status", "Message"},
	}

	for _, run := range c.store.Items {
		data = append(data, []string{
			run.CreatedAt.String(),
			run.ID,
			run.ConfigurationVersion.ID,
			run.Status,
			run.Message,
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
