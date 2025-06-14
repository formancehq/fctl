package stack

import (
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type EnableStore struct {
	Stack  *membershipclient.Stack `json:"stack"`
	Status string                  `json:"status"`
}
type EnableController struct {
	store *EnableStore
}

var _ fctl.Controller[*EnableStore] = (*EnableController)(nil)

func NewEnableStore() *EnableStore {
	return &EnableStore{
		Stack:  &membershipclient.Stack{},
		Status: "",
	}
}

func NewEnableController() *EnableController {
	return &EnableController{
		store: NewEnableStore(),
	}
}

func NewEnableCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)
	return fctl.NewMembershipCommand("enable (<stack-id> | --name=<stack-name>)",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Enable a stack"),
		fctl.WithArgs(cobra.MaximumNArgs(1)),
		fctl.WithStringFlag(stackNameFlag, "", "Stack to enable"),
		fctl.WithController(NewEnableController()),
	)
}
func (c *EnableController) GetStore() *EnableStore {
	return c.store
}

func (c *EnableController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	const (
		stackNameFlag = "name"
	)

	store := fctl.GetOrganizationStore(cmd)

	var stack *membershipclient.Stack
	if len(args) == 1 {
		if fctl.GetString(cmd, stackNameFlag) != "" {
			return nil, errors.New("need either an id of a name specified using --name flag")
		}

		rsp, _, err := store.Client().GetStack(cmd.Context(), store.OrganizationId(), args[0]).Execute()
		if err != nil {
			return nil, err
		}
		stack = rsp.Data
	} else {
		if fctl.GetString(cmd, stackNameFlag) == "" {
			return nil, errors.New("need either an id of a name specified using --name flag")
		}
		stacks, _, err := store.Client().ListStacks(cmd.Context(), store.OrganizationId()).Execute()
		if err != nil {
			return nil, errors.Wrap(err, "listing stacks")
		}
		for _, s := range stacks.Data {
			if s.Name == fctl.GetString(cmd, stackNameFlag) {
				stack = &s
				break
			}
		}
	}
	if stack == nil {
		return nil, errors.New("Stack not found")
	}

	if !fctl.CheckStackApprobation(cmd, stack, "You are about to enable stack '%s'", stack.Name) {
		return nil, fctl.ErrMissingApproval
	}

	if _, err := store.Client().EnableStack(cmd.Context(), store.OrganizationId(), stack.Id).Execute(); err != nil {
		return nil, errors.Wrap(err, "stack enable")
	}

	c.store.Stack = stack
	c.store.Status = "OK"

	return c, nil
}

func (c *EnableController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack enabled.")
	return nil
}
