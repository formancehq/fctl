package oauth_clients

import (
	"fmt"
	"strings"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/pointer"
	"github.com/spf13/cobra"
)

type Update struct {
	Client membershipclient.OrganizationClient `json:"organizationClient"`
}
type UpdateController struct {
	store *Update
}

var _ fctl.Controller[*Update] = (*UpdateController)(nil)

func NewDefaultUpdate() *Update {
	return &Update{}
}

func NewUpdateController() *UpdateController {
	return &UpdateController{
		store: NewDefaultUpdate(),
	}
}

func NewUpdateCommand() *cobra.Command {
	return fctl.NewCommand(`update <clientId>`,
		fctl.WithShortDescription("Update organization OAuth client"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag(descriptionFlag, "", "Description of the OAuth client usage"),
		fctl.WithStringFlag(nameFlag, "", "Name of the OAuth client"),
		fctl.WithController(NewUpdateController()),
	)
}

func (c *UpdateController) GetStore() *Update {
	return c.store
}

func (c *UpdateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("client_id is required")
	}
	description, err := cmd.Flags().GetString(descriptionFlag)
	if err != nil {
		return nil, err
	}

	name, err := cmd.Flags().GetString(nameFlag)
	if err != nil {
		return nil, err
	}

	clientId := args[0]
	clientId = strings.TrimPrefix(clientId, "organization_")
	if clientId == "" {
		return nil, fmt.Errorf("invalid client_id: %s", args[0])
	}

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	store, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to update an existing organization OAuth client") {
		return nil, fctl.ErrMissingApproval
	}

	actualClient, _, err := store.DefaultAPI.OrganizationClientRead(cmd.Context(), organizationID, clientId).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to read organization client: %w", err)
	}

	req := store.DefaultAPI.OrganizationClientUpdate(cmd.Context(), organizationID, clientId)
	reqBody := membershipclient.UpdateOrganizationClientRequest{}
	if description != "" {
		reqBody.Description = pointer.For(description)
	} else {
		reqBody.Description = pointer.For(actualClient.Data.Description)
	}

	if name != "" {
		reqBody.Name = name
	} else {
		reqBody.Name = actualClient.Data.Name
	}

	req = req.UpdateOrganizationClientRequest(reqBody)
	_, err = req.Execute()
	if err != nil {
		return nil, err
	}

	updatedClient, _, err := store.DefaultAPI.OrganizationClientRead(cmd.Context(), organizationID, clientId).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to read organization client: %w", err)
	}

	c.store.Client = updatedClient.Data

	return c, nil
}

func (c *UpdateController) Render(cmd *cobra.Command, args []string) error {
	return showOrganizationClient(cmd.OutOrStdout(), c.store.Client)
}
