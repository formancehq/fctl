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
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, err := fctl.LoadCurrentProfile(cmd, *cfg)
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
		profileName,
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
	app, err := apiClient.ReadApp(cmd.Context(), id)
	if err != nil {
		return nil, err
	}
	c.store.App = app.AppResponse.Data

	stateVersion, err := apiClient.ReadAppCurrentStateVersion(cmd.Context(), id)
	if err == nil {
		c.store.State = stateVersion.ReadStateResponse.Data
	}

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, args []string) error {

	if c.store.State.Stack != nil {
		cfg, err := fctl.LoadConfig(cmd)
		if err != nil {
			return err
		}

		profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
		if err != nil {
			return err
		}

		organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
		if err != nil {
			return err
		}

		membershipapiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
		if err != nil {
			return err
		}
		info, err := fctl.MembershipServerInfo(cmd.Context(), membershipapiClient)
		if err != nil {
			return err
		}

		if consoleURL := info.GetConsoleURL(); consoleURL != nil {
			pterm.Info.Printfln("View stack in console: %s/%s/%s?region=%s", *consoleURL, organizationID, c.store.State.Stack["id"], c.store.State.Stack["region_id"])
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
