package invitations

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type DeclineStore struct {
	Success      bool   `json:"success"`
	InvitationId string `json:"invitationId"`
}
type DeclineController struct {
	store *DeclineStore
}

var _ fctl.Controller[*DeclineStore] = (*DeclineController)(nil)

func NewDefaultDeclineStore() *DeclineStore {
	return &DeclineStore{}
}

func NewDeclineController() *DeclineController {
	return &DeclineController{
		store: NewDefaultDeclineStore(),
	}
}

func NewDeclineCommand() *cobra.Command {
	return fctl.NewCommand("decline <invitation-id>",
		fctl.WithAliases("dec", "d"),
		fctl.WithShortDescription("Decline invitation"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithController[*DeclineStore](NewDeclineController()),
	)
}

func (c *DeclineController) GetStore() *DeclineStore {
	return c.store
}

func (c *DeclineController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckOrganizationApprobation(cmd, "You are about to decline an invitation") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.DeclineInvitationRequest{
		InvitationID: args[0],
	}

	_, err = apiClient.DeclineInvitation(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.InvitationId = args[0]
	c.store.Success = true

	return c, nil
}

func (c *DeclineController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Invitation declined! %s", c.store.InvitationId)
	return nil
}
