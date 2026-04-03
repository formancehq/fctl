package deployments

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Deployments []components.Deployment
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
		fctl.WithShortDescription("List deployments for an app"),
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

	res, err := apiClient.ListDeployments(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store.Deployments = res.ListDeploymentsResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, _ []string) error {
	data := [][]string{
		{"Name", "App ID", "Stack ID", "Workspace ID"},
	}

	for _, d := range c.store.Deployments {
		data = append(data, []string{
			d.Name,
			d.AppID,
			d.StackID,
			d.WorkspaceID,
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
