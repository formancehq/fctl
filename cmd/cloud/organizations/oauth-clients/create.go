package oauth_clients

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/pointer"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
)

var (
	descriptionFlag = "description"
	nameFlag        = "name"
)

type Create struct {
	Client membershipclient.OrganizationClient `json:"organizationClient"`
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

	store := fctl.GetMembershipStore(cmd.Context())
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to create a new organization OAuth client") {
		return nil, fctl.ErrMissingApproval
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(descriptionFlag)
	if err != nil {
		return nil, err
	}

	name, err := cmd.Flags().GetString(nameFlag)
	if err != nil {
		return nil, err
	}

	req := store.Client().OrganizationClientCreate(cmd.Context(), organizationID)
	reqBody := membershipclient.CreateOrganizationClientRequest{}
	if description != "" {
		reqBody.Description = pointer.For(description)
	}

	if name != "" {
		reqBody.Name = pointer.For(name)
	}

	if description != "" || name != "" {
		req = req.CreateOrganizationClientRequest(reqBody)
	}

	response, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	c.store.Client = response.Data

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	return onCreateShow(cmd.OutOrStdout(), c.store.Client)
}
