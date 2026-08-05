package webhooks

import (
	"fmt"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowDeliveryStore struct {
	Delivery Delivery `json:"delivery"`
}

type ShowDeliveryController struct {
	store *ShowDeliveryStore
}

func NewShowDeliveryController() *ShowDeliveryController {
	return &ShowDeliveryController{store: &ShowDeliveryStore{}}
}

func (c *ShowDeliveryController) GetStore() *ShowDeliveryStore { return c.store }

func (c *ShowDeliveryController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	api, err := authenticatedDeliveriesAPI(cmd)
	if err != nil {
		return nil, err
	}
	response, err := api.get(cmd.Context(), args[0])
	if err != nil {
		return nil, fmt.Errorf("getting delivery: %w", err)
	}
	c.store.Delivery = response.Data
	return c, nil
}

func (c *ShowDeliveryController) Render(cmd *cobra.Command, _ []string) error {
	return renderDelivery(cmd, c.store.Delivery)
}

func NewShowDeliveryCommand() *cobra.Command {
	return fctl.NewCommand("show <delivery-id>",
		fctl.WithAliases("get"),
		fctl.WithShortDescription("Show a webhook delivery, including its payload"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*ShowDeliveryStore](NewShowDeliveryController()),
	)
}
