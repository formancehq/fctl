package users

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"
	"github.com/formancehq/go-libs/v3/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type LinkStore struct {
	Id      string `json:"id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}
type LinkController struct {
	store *LinkStore
}

var _ fctl.Controller[*LinkStore] = (*LinkController)(nil)

func NewDefaultLinkStore() *LinkStore {
	return &LinkStore{}
}

func NewLinkController() *LinkController {
	return &LinkController{
		store: NewDefaultLinkStore(),
	}
}

func NewLinkCommand() *cobra.Command {
	return fctl.NewCommand("link <user-id>",
		fctl.WithIntFlag("policy-id", 0, "Policy ID"),
		fctl.WithShortDescription("Link user to an organization with properties"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewLinkController()),
	)
}

func (c *LinkController) GetStore() *LinkStore {
	return c.store
}

func (c *LinkController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	policyID := fctl.GetInt(cmd, "policy-id")
	if policyID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}

	reqBody := components.UpdateOrganizationUserRequest{
		PolicyID: pointer.For(int64(policyID)),
	}

	request := operations.UpsertOrganizationUserRequest{
		OrganizationID: organizationID,
		UserID:         args[0],
		Body:           &reqBody,
	}

	response, err := apiClient.UpsertOrganizationUser(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	// Vérifier s'il y a une erreur dans la réponse
	if response.Error != nil {
		errMsg := "unknown error"
		if msg := response.Error.GetErrorMessage(); msg != nil {
			errMsg = *msg
		}
		return nil, fmt.Errorf("error updating user: %s", errMsg)
	}

	return c, nil
}

func (c *LinkController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("User Addd.")
	return nil
}
