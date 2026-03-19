package versions

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/go-libs/v3/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type List struct {
	components.ListVersionsResponseData
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ListVersionsResponseData: components.ListVersionsResponseData{},
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
		fctl.WithShortDescription("List versions for an app"),
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
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	versions, err := apiClient.ReadAppVersions(cmd.Context(), id, pointer.For(int64(fctl.GetInt(cmd, "page"))), pointer.For(int64(fctl.GetInt(cmd, "page-size"))))
	if err != nil {
		return nil, err
	}

	c.store.ListVersionsResponseData = versions.ListVersionsResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"ID", "AutoRunQueue", "Error", "ErrorMessage", "Status"},
	}

	for _, version := range c.store.Items {
		data = append(data, []string{
			string(version.ID),
			strconv.FormatBool(version.AutoQueueRuns),
			version.Error,
			version.ErrorMessage,
			string(version.Status),
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
