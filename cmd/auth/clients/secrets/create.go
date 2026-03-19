package secrets

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateStore struct {
	SecretId string `json:"secretId"`
	Name     string `json:"name"`
	Clear    string `json:"clear"`
}
type CreateController struct {
	store *CreateStore
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand("create <client-id> <secret-name>",
		fctl.WithAliases("c"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithShortDescription("Create secret"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*CreateStore](NewCreateController()),
	)
}

func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to create a new client secret") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.CreateSecretRequest{
		ClientID: args[0],
		CreateSecretRequest: &shared.CreateSecretRequest{
			Name:     args[1],
			Metadata: nil,
		},
	}
	response, err := stackClient.Auth.V1.CreateSecret(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.SecretId = response.CreateSecretResponse.Data.ID
	c.store.Name = response.CreateSecretResponse.Data.Name
	c.store.Clear = response.CreateSecretResponse.Data.Clear

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.SecretId})
	tableData = append(tableData, []string{pterm.LightCyan("Name"), c.store.Name})
	tableData = append(tableData, []string{pterm.LightCyan("Clear"), c.store.Clear})
	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
