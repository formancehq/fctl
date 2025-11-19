package oauth_clients

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

var (
	descriptionFlag = "description"
	nameFlag        = "name"
)

type Create struct {
	Client components.OrganizationClient `json:"organizationClient"`
}
type CreateController struct {
	store *Create
}

var _ fctl.Controller[*Create] = (*CreateController)(nil)

func NewDefaultCreate() *Create {
	return &Create{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultCreate(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand(`create`,
		fctl.WithShortDescription("Create organization OAuth client"),
		fctl.WithConfirmFlag(),
		fctl.WithStringFlag(descriptionFlag, "", "Description of the OAuth client usage"),
		fctl.WithStringFlag(nameFlag, "", "Name of the OAuth client"),
		fctl.WithController(NewCreateController()),
	)
}

func (c *CreateController) GetStore() *Create {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to create a new organization OAuth client") {
		return nil, fctl.ErrMissingApproval
	}

	description, err := cmd.Flags().GetString(descriptionFlag)
	if err != nil {
		return nil, err
	}

	name, err := cmd.Flags().GetString(nameFlag)
	if err != nil {
		return nil, err
	}

	reqBody := components.CreateOrganizationClientRequest{}
	if description != "" {
		reqBody.Description = pointer.For(description)
	}

	if name != "" {
		reqBody.Name = pointer.For(name)
	}

	request := operations.OrganizationClientCreateRequest{
		OrganizationID: organizationID,
		Body:           &reqBody,
	}

	response, err := apiClient.OrganizationClientCreate(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.CreateOrganizationClientResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Client = response.CreateOrganizationClientResponse.GetData()

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	return onCreateShow(cmd.OutOrStdout(), c.store.Client)
}
