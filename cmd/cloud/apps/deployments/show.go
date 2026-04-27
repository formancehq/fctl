package deployments

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Show struct {
	components.DeploymentResource
}

type ShowCtrl struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowCtrl)(nil)

func newShowStore() *Show {
	return &Show{}
}

func NewShowCtrl() *ShowCtrl {
	return &ShowCtrl{
		store: newShowStore(),
	}
}

func NewShow() *cobra.Command {
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show deployment details"),
		fctl.WithStringFlag("id", "", "Deployment ID"),
		fctl.WithBoolFlag("include-state", false, "Include Terraform state"),
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

	var includes []operations.ReadDeploymentInclude
	if fctl.GetBool(cmd, "include-state") {
		includes = append(includes, operations.ReadDeploymentIncludeState)
	}

	deployment, err := apiClient.ReadDeployment(cmd.Context(), id, includes)
	if err != nil {
		return nil, err
	}

	c.store.DeploymentResource = deployment.DeploymentResponse.Data

	return c, nil
}

func (c *ShowCtrl) Render(cmd *cobra.Command, _ []string) error {
	pterm.DefaultSection.Println("Deployment")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("ID: %s", c.store.ID)},
		{Level: 0, Text: fmt.Sprintf("App ID: %s", c.store.AppID)},
		{Level: 0, Text: fmt.Sprintf("Run Status: %s", c.store.RunStatus)},
		{Level: 0, Text: fmt.Sprintf("Created At: %s", c.store.CreatedAt)},
		{Level: 0, Text: fmt.Sprintf("Updated At: %s", c.store.UpdatedAt)},
	}

	if c.store.ManifestID != nil {
		items = append(items, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("Manifest ID: %s", *c.store.ManifestID)})
	}
	if c.store.ManifestVersion != nil {
		items = append(items, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("Manifest Version: %d", *c.store.ManifestVersion)})
	}
	if c.store.RunID != nil {
		items = append(items, pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("Run ID: %s", *c.store.RunID)})
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
