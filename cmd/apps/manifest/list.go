package manifest

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Versions []components.ManifestVersion
}

type ListCtrl struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListCtrl)(nil)

func newListStore() *ListStore {
	return &ListStore{}
}

func NewListCtrl() *ListCtrl {
	return &ListCtrl{
		store: newListStore(),
	}
}

func NewList() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls"),
		fctl.WithShortDescription("List manifest versions"),
		fctl.WithStringFlag("id", "", "App ID (required)"),
		fctl.WithController(NewListCtrl()),
	)
}

func (c *ListCtrl) GetStore() *ListStore {
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

	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("--id is required")
	}

	res, err := apiClient.ListManifestVersions(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store.Versions = res.ListManifestsResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, _ []string) error {
	data := [][]string{
		{"Version", "App ID", "Created At"},
	}

	for _, v := range c.store.Versions {
		data = append(data, []string{
			strconv.Itoa(v.Version),
			v.AppID,
			v.CreatedAt.String(),
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
