package secrets

import (
	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	return cmdbuilder.NewCommand("create [CLIENT_ID] [SECRET_NAME]",
		cmdbuilder.WithAliases("c"),
		cmdbuilder.WithArgs(cobra.ExactArgs(2)),
		cmdbuilder.WithShortDescription("Create secret"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			response, _, err := authClient.DefaultApi.
				CreateSecret(cmd.Context(), args[0]).
				Body(authclient.SecretOptions{
					Name:     args[1],
					Metadata: nil,
				}).
				Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("ID"), response.Data.Id})
			tableData = append(tableData, []string{pterm.LightCyan("Name"), response.Data.Name})
			tableData = append(tableData, []string{pterm.LightCyan("Clear"), response.Data.Clear})
			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
