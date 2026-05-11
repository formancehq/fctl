package variables

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/go-libs/v4/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type List struct {
	components.ReadVariablesResponseCursor
}

type ListCtrl struct {
	store *List
}

var _ fctl.Controller[*List] = (*ListCtrl)(nil)

func newDefaultStore() *List {
	return &List{
		ReadVariablesResponseCursor: components.ReadVariablesResponseCursor{},
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
		fctl.WithIntFlag("page-size", 100, "Page size"),
		fctl.WithStringFlag("cursor", "", "Opaque cursor token for the next page"),
		fctl.WithController(NewListCtrl()),
	)
}

func (c *ListCtrl) GetStore() *List {
	return c.store
}

func (c *ListCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	var cursor *string
	if v := fctl.GetString(cmd, "cursor"); v != "" {
		cursor = &v
	}

	vars, err := apiClient.ReadAppVariables(
		cmd.Context(),
		id,
		pointer.For(int64(fctl.GetInt(cmd, "page-size"))),
		cursor,
	)
	if err != nil {
		return nil, err
	}

	c.store.ReadVariablesResponseCursor = vars.ReadVariablesResponse.Cursor

	return c, nil
}

func (c *ListCtrl) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Id", "Key", "Value", "Description"},
	}

	for _, variable := range c.store.Data {
		data = append(data, []string{
			variable.ID,
			variable.Key,
			variable.Value,
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
