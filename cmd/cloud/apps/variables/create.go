package variables

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
)

type Create struct {
	components.Variable
}

type CreateCtrl struct {
	store *Create
}

var _ fctl.Controller[*Create] = (*CreateCtrl)(nil)

func newCreateStore() *Create {
	return &Create{
		Variable: components.Variable{},
	}
}

func NewCreateCtrl() *CreateCtrl {
	return &CreateCtrl{
		store: newCreateStore(),
	}
}

func NewCreate() *cobra.Command {
	return fctl.NewCommand("create",
		fctl.WithShortDescription("Create new variable for an app"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithStringFlag("key", "", "Variable key"),
		fctl.WithStringFlag("value", "", "Variable value"),
		fctl.WithStringFlag("description", "", "Variable description"),
		fctl.WithBoolFlag("sensitive", true, "Is variable sensitive"),
		fctl.WithStringFlag("category", "", "Variable category (env or terraform)"),
		fctl.WithController(NewCreateCtrl()),
	)
}

func (c *CreateCtrl) GetStore() *Create {
	return c.store
}

func (c *CreateCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	v, err := store.Cli.CreateAppVariable(cmd.Context(), fctl.GetString(cmd, "id"), components.CreateVariableRequest{
		Variable: components.VariableData{
			Key:         fctl.GetString(cmd, "key"),
			Value:       fctl.GetString(cmd, "value"),
			Description: func() *string { s := fctl.GetString(cmd, "description"); return &s }(),
			Sensitive:   fctl.GetBool(cmd, "sensitive"),
		},
	})
	if err != nil {
		return nil, err
	}

	c.store.Variable = v.CreateVariableResponse.Data

	return c, nil
}

func (c *CreateCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.DefaultSection.Println("Variable")

	items := []pterm.BulletListItem{
		{Level: 0, Text: fmt.Sprintf("ID: %s", c.store.Variable.ID)},
		{Level: 0, Text: fmt.Sprintf("Key: %s", c.store.Variable.Key)},
		{Level: 0, Text: fmt.Sprintf("Value: %s", func() string {
			if c.store.Variable.Sensitive {
				return "****"
			}
			return c.store.Variable.Value
		}())},
		{Level: 0, Text: fmt.Sprintf("Description: %s", func() string {
			if c.store.Variable.Description == nil {
				return "N/A"
			}
			return *c.store.Variable.Description
		}())},
	}

	if err := pterm.
		DefaultBulletList.
		WithItems(items).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}
	return nil
}
