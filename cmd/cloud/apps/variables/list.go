package variables

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/pointer"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
)

type List struct {
	components.ReadVariablesResponseData
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ReadVariablesResponseData: components.ReadVariablesResponseData{},
	}
}

func NewListCtrl() *ListCtrl {
	return &ListCtrl{
		store: newDefaultStore(),
	}
}

func NewList() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls"),
		fctl.WithShortDescription("List variables for an app"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithIntFlag("page", 1, "Page number"),
		fctl.WithIntFlag("page-size", 100, "Page size"),
		fctl.WithController(NewListCtrl()),
	)
}

func (c *ListCtrl) GetStore() *List {
	return c.store
}

func (c *ListCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	vars, err := store.Cli.ReadAppVariables(cmd.Context(), id, pointer.For(int64(fctl.GetInt(cmd, "page"))), pointer.For(int64(fctl.GetInt(cmd, "page-size"))))
	if err != nil {
		return nil, err
	}

	c.store.ReadVariablesResponseData = vars.ReadVariablesResponse.Data

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Id", "Key", "Value", "Description"},
	}

	for _, variable := range c.store.ReadVariablesResponseData.Items {
		data = append(data, []string{
			variable.ID,
			variable.Key,
			func() string {
				if variable.Sensitive {
					return "****"
				}
				return variable.Value
			}(),
			func() string {
				if variable.Description == nil {
					return ""
				}
				return *variable.Description
			}(),
		})
	}
	if err := pterm.
		DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(data).
		Render(); err != nil {
		return err
	}
	return nil
}
