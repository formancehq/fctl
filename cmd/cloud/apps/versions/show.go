package versions

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Show struct {
	components.ConfigurationVersion
}

type ShowCtrl struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowCtrl)(nil)

func newShowStore() *Show {
	return &Show{
		ConfigurationVersion: components.ConfigurationVersion{},
	}
}

func NewShowCtrl() *ShowCtrl {
	return &ShowCtrl{
		store: newShowStore(),
	}
}

func NewShow() *cobra.Command {
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show version"),
		fctl.WithStringFlag("id", "", "Version ID"),
		fctl.WithController(NewShowCtrl()),
	)
}

func (c *ShowCtrl) GetStore() *Show {
	return c.store
}

func (c *ShowCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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
	version, err := apiClient.ReadVersion(cmd.Context(), id)
	if err != nil {
		return nil, err
	}

	c.store.ConfigurationVersion = version.AppVersionResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.DefaultSection.Println("Version")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("Id: %s", c.store.ConfigurationVersion.ID)},
		{Level: 0, Text: fmt.Sprintf("AutoRunQueue: %t", c.store.ConfigurationVersion.AutoQueueRuns)},
		{Level: 0, Text: fmt.Sprintf("Status: %s", c.store.ConfigurationVersion.Status)},
		{Level: 0, Text: fmt.Sprintf("Error: %s", c.store.ConfigurationVersion.Error)},
		{Level: 0, Text: fmt.Sprintf("ErrorMessage: %s", c.store.ConfigurationVersion.ErrorMessage)},
	}

	if err := pterm.
		DefaultBulletList.
		WithItems(items).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}
	return nil
}
