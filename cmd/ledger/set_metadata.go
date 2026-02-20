package ledger

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type SetMetadataStore struct {
	Success bool `json:"success"`
}
type SetMetadataController struct {
	store *SetMetadataStore
}

var _ fctl.Controller[*SetMetadataStore] = (*SetMetadataController)(nil)

func NewDefaultSetMetadataStore() *SetMetadataStore {
	return &SetMetadataStore{}
}

func NewSetMetadataController() *SetMetadataController {
	return &SetMetadataController{
		store: NewDefaultSetMetadataStore(),
	}
}

func NewSetMetadataCommand() *cobra.Command {
	return fctl.NewCommand("set-metadata <ledger-name> [<key>=<value>...]",
		fctl.WithShortDescription("Set metadata on a ledger (Start from ledger v2 api)"),
		fctl.WithAliases("sm", "set-meta"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.MinimumNArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*SetMetadataStore](NewSetMetadataController()),
	)
}

func (c *SetMetadataController) GetStore() *SetMetadataStore {
	return c.store
}

func (c *SetMetadataController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	metadata, err := fctl.ParseMetadata(args[1:])
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to set a metadata on ledger %s", args[0]) {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.V2UpdateLedgerMetadataRequest{
		Ledger:      args[0],
		RequestBody: metadata,
	}
	response, err := stackClient.Ledger.V2.UpdateLedgerMetadata(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.Success = response.StatusCode == 204
	return c, nil
}

func (c *SetMetadataController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Metadata added!")
	return nil
}
