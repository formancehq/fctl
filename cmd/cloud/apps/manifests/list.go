package manifests

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/go-libs/v4/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type List struct {
	components.ListManifestsResponseCursor
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ListManifestsResponseCursor: components.ListManifestsResponseCursor{},
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
		fctl.WithShortDescription("List manifests"),
		fctl.WithIntFlag("page-size", 100, "Page size"),
		fctl.WithStringFlag("cursor", "", "Opaque cursor token for the next page"),
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

	var cursor *string
	if v := fctl.GetString(cmd, "cursor"); v != "" {
		cursor = &v
	}

	manifests, err := apiClient.ListManifests(
		cmd.Context(),
		pointer.For(int64(fctl.GetInt(cmd, "page-size"))),
		cursor,
	)
	if err != nil {
		return nil, err
	}

	c.store.ListManifestsResponseCursor = manifests.ListManifestsResponse.Cursor

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, _ []string) error {
	data := [][]string{
		{"ID", "Name", "Latest Version", "Created At"},
	}

	for _, m := range c.store.Data {
		data = append(data, []string{
			m.ID,
			m.Name,
			strconv.FormatInt(m.LatestVersion, 10),
			fmt.Sprint(m.CreatedAt),
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
