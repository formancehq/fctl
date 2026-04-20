package apps

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Show struct {
	components.App
}

type ShowCtrl struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowCtrl)(nil)

func newShowStore() *Show {
	return &Show{
		App: components.App{},
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

	app, err := apiClient.ReadApp(cmd.Context(), id, []operations.ReadAppInclude{operations.ReadAppIncludeState})
	if err != nil {
		return nil, err
	}
	c.store.App = app.AppResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, args []string) error {

	if c.store.App.State != nil && c.store.App.State.Stack != nil {
		_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
		if err != nil {
			return err
		}

		organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
		if err != nil {
			return err
		}
		info, err := fctl.MembershipServerInfo(cmd.Context(), apiClient)
		if err != nil {
			return err
		}

		if consoleURL := info.GetConsoleURL(); consoleURL != nil {
			pterm.Info.Printfln("View stack in console: %s/%s/%s?region=%s", *consoleURL, organizationID, c.store.App.State.Stack["id"], c.store.App.State.Stack["region_id"])
		}
	}

	pterm.DefaultSection.Println("App")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("ID: %s", c.store.App.ID)},
		{Level: 0, Text: fmt.Sprintf("Name: %s", c.store.App.Name)},
	}

	if c.store.App.StackID != nil {
		items = append(items, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("Stack ID: %s", *c.store.App.StackID)})
	}

	if err := pterm.
		DefaultBulletList.
		WithItems(items).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}

	if c.store.App.State != nil && c.store.App.State.Stack != nil {
		pterm.DefaultSection.Println("State")

		stateItems := []pterm.BulletListItem{}

		for k, v := range c.store.App.State.Stack {
			if v == nil {
				continue
			}
			stateItems = append(stateItems, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("%s: %s", k, v)})
		}

		slices.SortFunc(stateItems, func(a, b pterm.BulletListItem) int {
			return strings.Compare(a.Text, b.Text)
		})
		if err := pterm.
			DefaultBulletList.
			WithItems(stateItems).
			WithWriter(cmd.OutOrStdout()).
			Render(); err != nil {
			return err
		}
	}

	return nil
}
