package me

import (
	"errors"
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type InfoStore struct {
	Subject string `json:"subject"`
	Email   string `json:"email"`

	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}
type InfoController struct {
	store *InfoStore
}

var _ fctl.Controller[*InfoStore] = (*InfoController)(nil)

func NewDefaultInfoStore() *InfoStore {
	return &InfoStore{}
}

func NewInfoController() *InfoController {
	return &InfoController{
		store: NewDefaultInfoStore(),
	}
}

func NewInfoCommand() *cobra.Command {
	return fctl.NewCommand("info",
		fctl.WithAliases("i", "in"),
		fctl.WithShortDescription("Display user information"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithController(NewInfoController()),
	)
}

func (c *InfoController) GetStore() *InfoStore {
	return c.store
}

func (c *InfoController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetMembershipStore(cmd.Context())
	profile := fctl.GetCurrentProfile(cmd, store.Config)
	if !profile.IsConnected() {
		return nil, errors.New("not logged. use 'login' command before")
	}

	me, _, err := store.Client().ReadConnectedUser(cmd.Context()).Execute()
	if err != nil {
		return nil, err
	}

	userInfo, err := profile.GetUserInfo(cmd)
	if err != nil {
		return nil, err
	}

	c.store.Subject = userInfo.Subject
	c.store.Email = userInfo.Email
	c.store.ClientSecret = me.Data.ClientSecret
	c.store.ClientId = fmt.Sprintf("user_%s", me.Data.Id)

	return c, nil
}

func (c *InfoController) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Subject"), c.store.Subject})
	tableData = append(tableData, []string{pterm.LightCyan("Email"), c.store.Email})
	tableData = append(tableData, []string{pterm.LightCyan("Client ID"), c.store.ClientId})
	tableData = append(tableData, []string{pterm.LightCyan("Client Secret"), c.store.ClientSecret})

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
