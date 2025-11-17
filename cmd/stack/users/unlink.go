package users

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
)

type UnlinkStore struct {
	Stack  *membershipclient.Stack `json:"stack"`
	Status string                  `json:"status"`
}
type UnlinkController struct {
	store *UnlinkStore
}

var _ fctl.Controller[*UnlinkStore] = (*UnlinkController)(nil)

func NewDefaultUnlinkStore() *UnlinkStore {
	return &UnlinkStore{
		Stack:  &membershipclient.Stack{},
		Status: "",
	}
}

func NewUnlinkController() *UnlinkController {
	return &UnlinkController{
		store: NewDefaultUnlinkStore(),
	}
}

func NewUnlinkCommand() *cobra.Command {
	return fctl.NewMembershipCommand("unlink <user-id>",
		fctl.WithShortDescription("Unlink stack user within an organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewUnlinkController()),
	)
}
func (c *UnlinkController) GetStore() *UnlinkStore {
	return c.store
}

func (c *UnlinkController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetMembershipStackStore(cmd.Context())

	res, err := store.Client().DeleteStackUserAccess(cmd.Context(), store.OrganizationId(), store.StackId(), args[0]).Execute()
	if err != nil {
		return nil, err
	}

	if res.StatusCode > 300 {
		return nil, err
	}

	return c, nil
}

func (c *UnlinkController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack user access deleted.")
	return nil
}
