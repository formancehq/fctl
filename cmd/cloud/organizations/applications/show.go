package applications

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/time"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	Application *components.ApplicationWithScope `json:"application"`
}

type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewDefaultShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand(`show <application-id>`,
		fctl.WithAliases("s", "sh"),
		fctl.WithShortDescription("Show application details"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewShowController()),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	request := operations.GetOrganizationApplicationRequest{
		OrganizationID: organizationID,
		ApplicationID:  args[0],
	}

	response, err := apiClient.GetOrganizationApplication(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.GetApplicationResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Application = response.GetApplicationResponse.GetData()

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	if c.store.Application == nil {
		return fmt.Errorf("no application data")
	}

	app := c.store.Application

	data := [][]string{
		{"ID", app.GetID()},
		{"Name", app.GetName()},
		{"Alias", app.GetAlias()},
		{"URL", app.GetURL()},
		{"Description", func() string {
			if desc := app.GetDescription(); desc != nil {
				return *desc
			}
			return ""
		}()},
		{"Created At", time.Time{Time: app.GetCreatedAt()}.String()},
		{"Updated At", time.Time{Time: app.GetUpdatedAt()}.String()},
	}

	if scopes := app.GetScopes(); len(scopes) > 0 {
		data = append(data, []string{"Scopes", ""})
		for _, scope := range scopes {
			data = append(data, []string{
				"",
				fmt.Sprintf("  - %s (ID: %d)", scope.GetLabel(), scope.GetID()),
			})
			if desc := scope.GetDescription(); desc != nil && *desc != "" {
				data = append(data, []string{"", fmt.Sprintf("    Description: %s", *desc)})
			}
		}
	}

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(data).
		Render()
}
