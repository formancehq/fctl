package ledger

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v3/pointer"

	fctl "github.com/formancehq/fctl/pkg"
)

const (
	bucketNameFlag = "bucket"
	featuresFlag   = "features"
)

type CreateStore struct{}

type CreateController struct {
	store         *CreateStore
	metadataFlag  string
	referenceFlag string
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store:         NewDefaultCreateStore(),
		metadataFlag:  "metadata",
		referenceFlag: "reference",
	}
}

func NewCreateCommand() *cobra.Command {
	c := NewCreateController()
	return fctl.NewCommand("create <name>",
		fctl.WithAliases("c", "cr"),
		fctl.WithShortDescription("Create a new ledger (starting from ledger v2)"),
		fctl.WithStringFlag(bucketNameFlag, "", "Bucket on which install the new ledger"),
		fctl.WithStringSliceFlag(featuresFlag, []string{}, `Experimental! Features to enable on the newly created ledger (default: all)
Starting from ledger v2.2`),
		fctl.WithStringSliceFlag(c.metadataFlag, []string{}, "Metadata to apply on the newly created ledger"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*CreateStore](c),
	)
}

func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())
	if !fctl.CheckStackApprobation(cmd, store.Stack(), "You are about to create a new ledger") {
		return nil, fctl.ErrMissingApproval
	}

	metadata, err := fctl.ParseMetadata(fctl.GetStringSlice(cmd, c.metadataFlag))
	if err != nil {
		return nil, err
	}

	features := make(map[string]string)
	for _, s := range fctl.GetStringSlice(cmd, featuresFlag) {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid feature flag format")
		}
		features[parts[0]] = parts[1]
	}

	_, err = store.Client().Ledger.V2.CreateLedger(cmd.Context(), operations.V2CreateLedgerRequest{
		V2CreateLedgerRequest: shared.V2CreateLedgerRequest{
			Bucket:   pointer.For(fctl.GetString(cmd, bucketNameFlag)),
			Metadata: metadata,
			Features: features,
		},
		Ledger: args[0],
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Ledger created!")
	return nil
}
