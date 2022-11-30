package secrets

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	return internal2.NewCommand("create [CLIENT_ID] [SECRET_NAME]",
		internal2.WithAliases("c"),
		internal2.WithArgs(cobra.ExactArgs(2)),
		internal2.WithShortDescription("Create secret"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			authClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := authClient.ClientsApi.
				CreateSecret(cmd.Context(), args[0]).
				Body(formance.SecretOptions{
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
