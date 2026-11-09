package connections

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type SuspensionStore struct {
	Name    string `json:"name"`
	Suspend bool   `json:"suspend"`
	// Connection is the connection returned by the PATCH.
	Connection *connectivity.Connection `json:"connection,omitempty"`
}

type SuspensionController struct {
	approve approvalFunc
	store   *SuspensionStore
	suspend bool
}

var _ fctl.Controller[*SuspensionStore] = (*SuspensionController)(nil)

func NewSuspensionController(suspend bool) *SuspensionController {
	return &SuspensionController{
		approve: fctl.CheckStackApprobation,
		store:   &SuspensionStore{},
		suspend: suspend,
	}
}

func NewSuspendCommand() *cobra.Command {
	return newSuspensionCommand("suspend", "Pause ingestion for a connection", true)
}

func NewUnsuspendCommand() *cobra.Command {
	return newSuspensionCommand("unsuspend", "Resume ingestion for a connection", false)
}

func newSuspensionCommand(name, description string, suspend bool) *cobra.Command {
	controller := NewSuspensionController(suspend)
	return fctl.NewCommand(
		name+" <connection>",
		fctl.WithShortDescription(description),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectionNames()),
		fctl.WithConfirmFlag(),
		fctl.WithController[*SuspensionStore](controller),
		fctl.WithFlagUsage("confirm", "Skip the confirmation prompt"),
		fctl.WithExamples("fctl connectivity connections "+name+" stripeconnect-fpss --confirm"),
	)
}

func (c *SuspensionController) GetStore() *SuspensionStore {
	return c.store
}

func (c *SuspensionController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	client, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, client.Connectivity.V1)
}

func (c *SuspensionController) run(cmd *cobra.Command, args []string, client *sdk.FormanceV1) (fctl.Renderable, error) {
	name := args[0]
	action := "suspend"
	if !c.suspend {
		action = "resume"
	}
	if !c.approve(cmd, "You are about to %s Connectivity connection %q", action, name) {
		return nil, fctl.ErrMissingApproval
	}
	response, err := client.PatchConnection(cmd.Context(), operations.PatchConnectionRequest{Name: url.PathEscape(name), ConnectionPatch: connectivity.ConnectionPatch{Configuration: &connectivity.Configuration{Suspend: &c.suspend}}}, nil)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Connection == nil || response.Connection.Name == "" {
		return nil, fmt.Errorf("suspend connectivity connection %q: empty response", name)
	}
	c.store.Name = name
	c.store.Suspend = c.suspend
	c.store.Connection = response.Connection
	return c, nil
}

func (c *SuspensionController) Render(cmd *cobra.Command, _ []string) error {
	request := "suspension"
	if !c.store.Suspend {
		request = "resumption"
	}
	return fctl.RenderText(cmd.OutOrStdout(), fmt.Sprintf("Connection %q %s requested.", c.store.Name, request))
}
