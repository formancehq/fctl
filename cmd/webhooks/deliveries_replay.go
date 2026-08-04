package webhooks

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ReplayDeliveryStore struct {
	Delivery Delivery `json:"delivery"`
}

type ReplayDeliveryController struct {
	store *ReplayDeliveryStore
}

func NewReplayDeliveryController() *ReplayDeliveryController {
	return &ReplayDeliveryController{store: &ReplayDeliveryStore{}}
}

func (c *ReplayDeliveryController) GetStore() *ReplayDeliveryStore { return c.store }

func (c *ReplayDeliveryController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if !fctl.CheckStackApprobation(cmd, "You are about to replay webhook delivery %s", args[0]) {
		return nil, fctl.ErrMissingApproval
	}
	api, err := authenticatedDeliveriesAPI(cmd)
	if err != nil {
		return nil, err
	}
	response, err := api.replay(cmd.Context(), args[0], fctl.GetString(cmd, idempotencyKeyFlag))
	if err != nil {
		return nil, fmt.Errorf("replaying delivery: %w", err)
	}
	c.store.Delivery = response.Data
	return c, nil
}

func (c *ReplayDeliveryController) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Delivery queued for replay")
	return renderDelivery(cmd, c.store.Delivery)
}

func NewReplayDeliveryCommand() *cobra.Command {
	cmd := fctl.NewCommand("replay <delivery-id>",
		fctl.WithShortDescription("Replay one failed or pending webhook delivery"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(idempotencyKeyFlag, "", "Idempotency key for this replay request"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*ReplayDeliveryStore](NewReplayDeliveryController()),
	)
	if err := cmd.MarkFlagRequired(idempotencyKeyFlag); err != nil {
		panic(err)
	}
	return cmd
}
