package clients

import (
	"fmt"

	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show [CLIENT_ID]",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithAliases("s"),
		cmdbuilder.WithShortDescription("Show client"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			response, _, err := authClient.DefaultApi.ReadClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("ID"), response.Data.Id})
			tableData = append(tableData, []string{pterm.LightCyan("Name"), response.Data.Name})
			tableData = append(tableData, []string{pterm.LightCyan("Description"), cmdbuilder.StringPointerToString(response.Data.Description)})
			tableData = append(tableData, []string{pterm.LightCyan("Public"), cmdbuilder.BoolPointerToString(response.Data.Public)})

			cmdbuilder.Highlightln(cmd.OutOrStdout(), "Information :")
			if err := pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "")

			if len(response.Data.RedirectUris) > 0 {
				cmdbuilder.Highlightln(cmd.OutOrStdout(), "Redirect URIs :")
				if err := pterm.DefaultBulletList.WithWriter(cmd.OutOrStdout()).WithItems(collections.Map(response.Data.RedirectUris, func(redirectURI string) pterm.BulletListItem {
					return pterm.BulletListItem{
						Text:        redirectURI,
						TextStyle:   pterm.NewStyle(pterm.FgDefault),
						BulletStyle: pterm.NewStyle(pterm.FgLightCyan),
					}
				})).Render(); err != nil {
					return err
				}
			}

			if len(response.Data.PostLogoutRedirectUris) > 0 {
				cmdbuilder.Highlightln(cmd.OutOrStdout(), "Post logout redirect URIs :")
				if err := pterm.DefaultBulletList.WithWriter(cmd.OutOrStdout()).WithItems(collections.Map(response.Data.PostLogoutRedirectUris, func(redirectURI string) pterm.BulletListItem {
					return pterm.BulletListItem{
						Text:        redirectURI,
						TextStyle:   pterm.NewStyle(pterm.FgDefault),
						BulletStyle: pterm.NewStyle(pterm.FgLightCyan),
					}
				})).Render(); err != nil {
					return err
				}
			}

			if len(response.Data.Secrets) > 0 {
				cmdbuilder.Highlightln(cmd.OutOrStdout(), "Secrets :")

				if err := pterm.DefaultTable.
					WithWriter(cmd.OutOrStdout()).
					WithHasHeader(true).
					WithData(collections.Prepend(
						collections.Map(response.Data.Secrets, func(secret authclient.ClientSecret) []string {
							return []string{
								secret.Id, secret.Name, secret.LastDigits,
							}
						}),
						[]string{"ID", "Name", "Last digits"},
					)).
					Render(); err != nil {
					return err
				}
			}

			return nil
		}),
	)
}
