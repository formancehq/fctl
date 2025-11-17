package apps

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
)

type Show struct {
	components.App
	State components.State
}

type ShowCtrl struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowCtrl)(nil)

func newShowStore() *Show {
	return &Show{
		App:   components.App{},
		State: components.State{},
	}
}

func NewShowCtrl() *ShowCtrl {
	return &ShowCtrl{
		store: newShowStore(),
	}
}

func NewShow() *cobra.Command {
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show apps"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithController(NewShowCtrl()),
	)
}

func (c *ShowCtrl) GetStore() *Show {
	return c.store
}

func (c *ShowCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	app, err := store.Cli.ReadApp(cmd.Context(), id)
	if err != nil {
		return nil, err
	}
	c.store.App = app.AppResponse.Data

	stateVersion, err := store.Cli.ReadAppCurrentStateVersion(cmd.Context(), id)
	if err == nil {
		c.store.State = stateVersion.ReadStateResponse.Data
	}

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, args []string) error {

	if c.store.State.Stack != nil {
		cfg, err := fctl.GetConfig(cmd)
		if err != nil {
			return err
		}
		membershipStore := fctl.GetMembershipStore(cmd.Context())
		organizationID, err := fctl.ResolveOrganizationID(cmd, cfg, membershipStore.Client())
		if err != nil {
			return err
		}
		info, _, err := membershipStore.Client().GetServerInfo(cmd.Context()).Execute()
		if err != nil {
			return err
		}

		if info.ConsoleURL != nil {
			pterm.Info.Printfln("View stack in console: %s/%s/%s?region=%s", *info.ConsoleURL, organizationID, c.store.State.Stack["id"], c.store.State.Stack["region_id"])
		}
	}

	pterm.DefaultSection.Println("App")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("ID: %s", c.store.App.ID)},
		{Level: 0, Text: fmt.Sprintf("Name: %s", c.store.App.Name)},
		{Level: 0, Text: fmt.Sprintf("Run Status: %s", func() string {
			if c.store.App.CurrentRun == nil {
				return "N/A"
			}
			return c.store.App.CurrentRun.Status
		}())},
	}

	if err := pterm.
		DefaultBulletList.
		WithItems(items).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}

	if c.store.State.Stack != nil {
		pterm.DefaultSection.Println("State")

		items = []pterm.BulletListItem{}

		for k, v := range c.store.State.Stack {
			if v == nil {
				continue
			}
			items = append(items, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("%s: %s", k, v)})
		}

		slices.SortFunc(items, func(a, b pterm.BulletListItem) int {
			return strings.Compare(a.Text, b.Text)
		})
		if err := pterm.
			DefaultBulletList.
			WithItems(items).
			WithWriter(cmd.OutOrStdout()).
			Render(); err != nil {
			return err
		}
	}

	return nil
}
