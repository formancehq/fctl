package oauth_clients

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Update struct {
	Client components.OrganizationClient `json:"organizationClient"`
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to update an existing organization OAuth client") {
		return nil, fctl.ErrMissingApproval
	}

	readRequest := operations.OrganizationClientReadRequest{
		OrganizationID: organizationID,
		ClientID:       clientId,
	}

	actualClientResponse, err := apiClient.OrganizationClientRead(cmd.Context(), readRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to read organization client: %w", err)
	}

	if actualClientResponse.ReadOrganizationClientResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	actualClient := actualClientResponse.ReadOrganizationClientResponse.GetData()

	reqBody := components.UpdateOrganizationClientRequest{}
	if description != "" {
		reqBody.Description = pointer.For(description)
	} else {
		reqBody.Description = pointer.For(actualClient.GetDescription())
	}

	if name != "" {
		reqBody.Name = name
	} else {
		reqBody.Name = actualClient.GetName()
	}

	updateRequest := operations.OrganizationClientUpdateRequest{
		OrganizationID: organizationID,
		ClientID:       clientId,
		Body:           &reqBody,
	}

	_, err = apiClient.OrganizationClientUpdate(cmd.Context(), updateRequest)
	if err != nil {
		return nil, err
	}

	updatedClientResponse, err := apiClient.OrganizationClientRead(cmd.Context(), readRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to read organization client: %w", err)
	}

	if updatedClientResponse.ReadOrganizationClientResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Client = updatedClientResponse.ReadOrganizationClientResponse.GetData()

	return c, nil
}

func (c *UpdateController) Render(cmd *cobra.Command, args []string) error {
	return showOrganizationClient(cmd.OutOrStdout(), c.store.Client)
}
