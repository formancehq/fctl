package ledger

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	response, err := stackClient.Ledger.V1.GetInfo(cmd.Context())
	if err != nil {
		return nil, err
	}

	c.store.Server = response.ConfigInfoResponse.Data.Server
	c.store.Version = response.ConfigInfoResponse.Data.Version

	return c, nil
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
