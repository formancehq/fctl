package clients

import (
	"fmt"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return internal2.NewCommand("show [CLIENT_ID]",
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithAliases("s"),
		internal2.WithShortDescription("Show client"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			authClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := authClient.ClientsApi.ReadClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("ID"), response.Data.Id})
			tableData = append(tableData, []string{pterm.LightCyan("Name"), response.Data.Name})
			tableData = append(tableData, []string{pterm.LightCyan("Description"), internal2.StringPointerToString(response.Data.Description)})
			tableData = append(tableData, []string{pterm.LightCyan("Public"), internal2.BoolPointerToString(response.Data.Public)})

			internal2.Highlightln(cmd.OutOrStdout(), "Information :")
			if err := pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "")

			if len(response.Data.RedirectUris) > 0 {
				internal2.Highlightln(cmd.OutOrStdout(), "Redirect URIs :")
				if err := pterm.DefaultBulletList.WithWriter(cmd.OutOrStdout()).WithItems(internal2.Map(response.Data.RedirectUris, func(redirectURI string) pterm.BulletListItem {
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
				internal2.Highlightln(cmd.OutOrStdout(), "Post logout redirect URIs :")
				if err := pterm.DefaultBulletList.WithWriter(cmd.OutOrStdout()).WithItems(internal2.Map(response.Data.PostLogoutRedirectUris, func(redirectURI string) pterm.BulletListItem {
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
				internal2.Highlightln(cmd.OutOrStdout(), "Secrets :")

				if err := pterm.DefaultTable.
					WithWriter(cmd.OutOrStdout()).
					WithHasHeader(true).
					WithData(internal2.Prepend(
						internal2.Map(response.Data.Secrets, func(secret formance.ClientSecret) []string {
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
