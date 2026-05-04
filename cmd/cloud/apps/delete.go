package apps

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Delete struct {
	ID                  string  `json:"id"`
	DestroyDeploymentID *string `json:"destroyDeploymentId,omitempty"`
	Waited              bool    `json:"waited"`
}

type DeleteCtrl struct {
	store *Delete
}

var _ fctl.Controller[*Delete] = (*DeleteCtrl)(nil)

func newDeleteStore() *Delete {
	return &Delete{}
}

func NewDeleteCtrl() *DeleteCtrl {
	return &DeleteCtrl{
		store: newDeleteStore(),
	}
}

func NewDelete() *cobra.Command {
	return fctl.NewCommand("delete",
		fctl.WithShortDescription("Soft-delete an app and enqueue a destroy deployment"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithBoolFlag("wait", false, "Block until the destroy deployment reaches a terminal status"),
		fctl.WithController(NewDeleteCtrl()),
	)
}

func (c *DeleteCtrl) GetStore() *Delete {
	return c.store
}

func (c *DeleteCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	wait := fctl.GetBool(cmd, "wait")
	resp, err := apiClient.DeleteApp(cmd.Context(), id, &wait)
	if err != nil {
		return nil, err
	}

	c.store.ID = id
	c.store.Waited = wait
	if resp.DeleteAppResponse != nil {
		c.store.DestroyDeploymentID = resp.DeleteAppResponse.DestroyDeploymentID
	}

	return c, nil
}

func (c *DeleteCtrl) Render(cmd *cobra.Command, args []string) error {
	switch {
	case c.store.DestroyDeploymentID != nil && *c.store.DestroyDeploymentID != "":
		if c.store.Waited {
			pterm.Success.Printfln("App %s deleted (destroy deployment %s reached terminal status)",
				c.store.ID, *c.store.DestroyDeploymentID)
		} else {
			pterm.Success.Printfln("App %s soft-deleted; destroy deployment %s enqueued (poll with `fctl cloud app deployments show --id %s`)",
				c.store.ID, *c.store.DestroyDeploymentID, *c.store.DestroyDeploymentID)
		}
	default:
		pterm.Success.Printfln("App %s deleted (no destroy deployment was needed)", c.store.ID)
	}
	return nil
}
