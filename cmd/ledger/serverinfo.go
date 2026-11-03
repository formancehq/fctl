package ledger

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
)

type ServerInfoStore struct {
	Server  string `json:"server"`
	Version string `json:"version"`
}
type ServerInfoController struct {
	store *ServerInfoStore
}

var _ fctl.Controller[*ServerInfoStore] = (*ServerInfoController)(nil)

func NewDefaultServerInfoStore() *ServerInfoStore {
	return &ServerInfoStore{
		Server:  "unknown",
		Version: "unknown",
	}
}

func NewServerInfoController() *ServerInfoController {
	return &ServerInfoController{
		store: NewDefaultServerInfoStore(),
	}
}

func NewServerInfoCommand() *cobra.Command {
	return fctl.NewCommand("server-infos",
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithAliases("si"),
		fctl.WithShortDescription("Read server info"),
		fctl.WithController[*ServerInfoStore](NewServerInfoController()),
	)
}

func (c *ServerInfoController) GetStore() *ServerInfoStore {
	return c.store
}

func (c *ServerInfoController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := c.readInfo(cmd.Context(), stackClient); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ServerInfoController) readInfo(ctx context.Context, stackClient *formance.Formance) error {
	response, err := stackClient.Ledger.V2.GetInfo(ctx)
	if err != nil {
		return err
	}

	if response == nil || response.V2ConfigInfo == nil {
		return fmt.Errorf("read Ledger server info: empty response")
	}
	c.store.Server = response.V2ConfigInfo.Server
	c.store.Version = response.V2ConfigInfo.Version

	return nil
}

func (c *ServerInfoController) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Server"), fmt.Sprint(c.store.Server)})
	tableData = append(tableData, []string{pterm.LightCyan("Version"), fmt.Sprint(c.store.Version)})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return nil
}
