package invitations

import (
	"fmt"

	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type InvitationSend struct {
	Email string `json:"email"`
}

type SendStore struct {
	Invitation InvitationSend `json:"invitation"`
}
type SendController struct {
	store *SendStore
}

var _ fctl.Controller[*SendStore] = (*SendController)(nil)

func NewDefaultSendStore() *SendStore {
	return &SendStore{
		Invitation: InvitationSend{},
	}
}

func NewSendController() *SendController {
	return &SendController{
		store: NewDefaultSendStore(),
	}
}

func NewSendCommand() *cobra.Command {
	return fctl.NewCommand("send <email>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithShortDescription("Invite a user by email"),
		fctl.WithAliases("s"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*SendStore](NewSendController()),
	)
}

func (c *SendController) GetStore() *SendStore {
	return c.store
}

func (c *SendController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to send an invitation") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.CreateInvitationRequest{
		OrganizationID: organizationID,
		Email:          args[0],
	}

	response, err := apiClient.CreateInvitation(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.CreateInvitationResponse == nil || response.CreateInvitationResponse.GetData() == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Invitation.Email = response.CreateInvitationResponse.GetData().GetUserEmail()

	return c, nil
}

func (c *SendController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Invitation sent to %s", c.store.Invitation.Email)
	return nil

}
